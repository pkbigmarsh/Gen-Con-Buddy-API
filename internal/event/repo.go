package event

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
	"github.com/rs/zerolog"
)

const (
	bulkMetaFormat = `{ "%s": { "_index": "%s", "_id": "%s" }`
	bulkIndexMeta  = `{ "index": { "_index": "%s", "_id": "%s" } }`
)

// EventRepo controls talking to the OpenSearch cluster for event details
type EventRepo struct {
	logger     *zerolog.Logger
	client     *opensearch.Client
	batchSize  int
	eventIndex string
}

// NewEventRepo instantiates a new EventRepo
func NewEventRepo(logger *zerolog.Logger, client *opensearch.Client, batchSize int, eventIndex string) *EventRepo {
	return &EventRepo{
		logger:     logger,
		client:     client,
		batchSize:  100,
		eventIndex: eventIndex,
	}
}

func (r *EventRepo) WriteEvents(ctx context.Context, events []*Event) ([]error, error) {
	r.logger.Info().Msgf("Writing %d events, expected batch count: %d", len(events), len(events)/r.batchSize)
	var (
		body      = ""
		batchSize = 0
		docErrs   []error
	)
	for _, e := range events {
		body += fmt.Sprintf(bulkIndexMeta, r.eventIndex, e.GameID) + "\n"
		docJson, err := json.Marshal(e)
		if err != nil {
			docErrs = append(docErrs, fmt.Errorf("failed to marshal event %s: %w", e.GameID, err))
			continue
		}

		body += string(docJson) + "\n"
		batchSize++

		if batchSize >= r.batchSize {
			errs, err := r.writeEvents(ctx, body)
			docErrs = append(docErrs, errs...)
			if err != nil {
				return docErrs, fmt.Errorf("failed to execute event write batch: %s", err)
			}

			body = ""
			batchSize = 0
		}
	}

	if batchSize > 0 {
		errs, err := r.writeEvents(ctx, body)
		docErrs = append(docErrs, errs...)
		return docErrs, err
	}

	return docErrs, nil
}

func (r *EventRepo) writeEvents(ctx context.Context, body string) ([]error, error) {
	r.logger.Debug().Msgf("Writing events with request: %s", body)
	bulkWriteReq := opensearchapi.BulkRequest{
		Index: r.eventIndex,
		Body:  strings.NewReader(body),
	}

	resp, err := bulkWriteReq.Do(ctx, r.client)
	if err != nil {
		return nil, err
	}

	defer func() {
		if resp.Body != nil {
			resp.Body.Close()
		}
	}()

	r.logger.Debug().Msgf("Raw write events response: %s", resp.String())

	if resp.IsError() {
		return nil, fmt.Errorf("failted to write some events, code [%d] headers [%v]", resp.StatusCode, resp.Header)
	}

	return nil, nil
}

func (r *EventRepo) Search(ctx context.Context, req SearchRequest) (SearchResponse, error) {
	r.logger.Debug().Msgf("performing search request: %+v", req)
	if req.Limit <= 0 {
		return SearchResponse{}, fmt.Errorf("limit cannot be less than 1, got %d", req.Limit)
	}

	if req.Page < 0 {
		return SearchResponse{}, fmt.Errorf("page must be non negative, got %d", req.Page)
	}

	searchBody := map[string]any{
		"track_total_hits": true,
		"size":             req.Limit,
		"from":             req.Limit * req.Page,
	}

	if len(req.Terms) != 0 {
		var must []any
		var termErrors []error
		for _, t := range req.Terms {
			query, err := t.ToQuery()
			if err != nil {
				termErrors = append(termErrors, err)
				continue
			}

			must = append(must, query)
		}

		if len(termErrors) > 0 {
			return SearchResponse{}, errors.Join(termErrors...)
		}

		searchBody["query"] = map[string]any{
			"bool": map[string]any{"must": must},
		}
	}

	bodyBytes, err := json.Marshal(searchBody)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("failed to marshal search request: %w", err)
	}

	r.logger.Debug().Msgf("Performing search request: %s", bodyBytes)

	osReq := opensearchapi.SearchRequest{
		Index: []string{r.eventIndex},
		Body:  bytes.NewReader(bodyBytes),
	}

	osResp, err := osReq.Do(ctx, r.client)
	if err != nil {
		return SearchResponse{}, err
	}
	defer func() {
		err := osResp.Body.Close()
		if err != nil {
			r.logger.Err(err).Msgf("failed to close search response body")
		}
	}()

	if osResp.IsError() {
		r.logger.Error().Msgf("search request failed. Raw response: %s", osResp.String())
		return SearchResponse{}, fmt.Errorf("faile search request %d", osResp.StatusCode)
	}

	r.logger.Debug().Msgf("search request debuging: %s", osResp.String())

	var (
		response eventSearchResponse
		events   []*Event
		buff     = bytes.NewBuffer([]byte{})
	)

	if _, err := buff.ReadFrom(osResp.Body); err != nil {
		return SearchResponse{}, fmt.Errorf("failed to read search response body: %w", err)
	}

	if err := json.Unmarshal(buff.Bytes(), &response); err != nil {
		return SearchResponse{}, fmt.Errorf("failed to unmarshal search response: %w", err)
	}

	for _, e := range response.Hits.Hits {
		events = append(events, e.Event)
	}

	return SearchResponse{
		TotalEvents: response.Hits.Total.Value,
		Events:      events,
	}, nil
}

type eventSearchResponse struct {
	Hits struct {
		Total struct {
			Value int64 `json:"value"`
		} `json:"total"`
		MaxScore float64 `json:"max_score"`
		Hits     []struct {
			Index string  `json:"_index"`
			ID    string  `json:"_id"`
			Score float64 `json:"_score"`
			Event *Event  `json:"_source,omitempty"`
		} `json:"hits"`
	} `json:"hits"`
	Errors bool `json:"errors"`
	Items  []struct {
		IndexError *struct {
			Index  string   `json:"_index"`
			ID     string   `json:"_id"`
			Status int      `json:"status"`
			Error  *osError `json:"error"`
		} `json:"index,omitempty"`
	} `json:"items,omitempty"`
}

type osError struct {
	Type      string `json:"type"`
	Reason    string `json:"reason"`
	Index     string `json:"index"`
	IndexUUID string `json:"index_uuid"`
}
