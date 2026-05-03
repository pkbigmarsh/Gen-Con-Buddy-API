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
	bulkMeta     = `{ "%s": { "_index": "%s", "_id": "%s" } }`
	createAction = "create"
	updateAction = "update"
)

// textSortFields are fields stored as OpenSearch `text` type.
// Sorting on them requires the `.keyword` subfield for lexicographic ordering.
var textSortFields = map[Field]struct{}{
	Group:                    {},
	Title:                    {},
	ShortDescription:         {},
	LongDescription:          {},
	GameSystem:               {},
	RulesEdition:             {},
	MaterialsProvided:        {},
	MaterialsRequiredDetails: {},
	GMNames:                  {},
	Tournament:               {},
	Location:                 {},
	RoomName:                 {},
	TableNumber:              {},
	Prize:                    {},
	RulesComplexity:          {},
}

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
		batchSize:  batchSize,
		eventIndex: eventIndex,
	}
}

func (r *EventRepo) CreateEvents(ctx context.Context, events []*Event) ([]error, error) {
	return r.WriteEvents(ctx, createAction, events)
}

func (r *EventRepo) WriteEvents(ctx context.Context, action string, events []*Event) ([]error, error) {
	r.logger.Info().Msgf("Writing %d events, expected batch count: %d", len(events), len(events)/r.batchSize)
	var (
		body      = ""
		batchSize = 0
		docErrs   []error
	)
	for _, e := range events {
		body += fmt.Sprintf(bulkMeta, action, r.eventIndex, e.GameID) + "\n"
		var (
			docJson []byte
			err     error
		)

		switch action {
		case createAction:
			docJson, err = json.Marshal(e)
		case updateAction:
			docJson, err = json.Marshal(bulkUpdateAction{Doc: e})
		default:
			return nil, fmt.Errorf("upsupported write action [%s]", action)
		}

		if err != nil {
			docErrs = append(docErrs, fmt.Errorf("failed to marshal event %s: %w", e.GameID, err))
			continue
		}

		body += string(docJson) + "\n"
		batchSize++

		if batchSize >= r.batchSize {
			errs, err := r.writeEvents(ctx, action, body)
			docErrs = append(docErrs, errs...)
			if err != nil {
				return docErrs, fmt.Errorf("failed to execute event write batch: %s", err)
			}

			body = ""
			batchSize = 0
		}
	}

	if batchSize > 0 {
		errs, err := r.writeEvents(ctx, action, body)
		docErrs = append(docErrs, errs...)
		return docErrs, err
	}

	return docErrs, nil
}

func (r *EventRepo) writeEvents(ctx context.Context, action string, body string) ([]error, error) {
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

	var (
		response bulkResponse
		buff     = bytes.NewBuffer([]byte{})
	)

	if _, err := buff.ReadFrom(resp.Body); err != nil {
		return nil, fmt.Errorf("failed to read bulk response body: %w", err)
	}

	if err := json.Unmarshal(buff.Bytes(), &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal bulk action response: %s", err)
	}

	if !response.Errors {
		// short circuit because there was no failures
		return nil, nil
	}

	var errs []error

	for _, i := range response.Items {
		actionResponse, ok := i[action]
		if !ok {
			idx := 0
			keys := make([]string, len(i))
			for k := range i {
				keys[idx] = k
				idx++
			}
			errs = append(errs, fmt.Errorf("no response for action [%s] found in response map, keys [%v]", action, keys))
		}
		r.logger.Debug().Msgf("response item: [%+v]", i)
		if actionResponse.Error != nil {
			errs = append(errs, fmt.Errorf("%s: %s", actionResponse.Error.Type, actionResponse.Error.Reason))
		}
	}

	return errs, nil
}

func (r *EventRepo) UpdateEvents(ctx context.Context, events []*Event) ([]error, error) {
	return r.WriteEvents(ctx, updateAction, events)
}

func (r *EventRepo) Search(ctx context.Context, req SearchRequest) (SearchResponse, error) {
	r.logger.Debug().Msgf("performing search request: %+v", req)
	if req.Limit <= 0 {
		return SearchResponse{}, fmt.Errorf("limit cannot be less than 1, got %d", req.Limit)
	}

	if req.Page < 0 {
		return SearchResponse{}, fmt.Errorf("page must be non negative, got %d", req.Page)
	}

	sortField := "startDateTime"
	sortDir := "asc"
	if req.SortField != "" {
		sortField = string(req.SortField)
		sortDir = req.SortDir
		if _, isText := textSortFields[req.SortField]; isText {
			sortField = sortField + ".keyword"
		}
	}

	searchBody := map[string]any{
		"track_total_hits": true,
		"size":             req.Limit,
		"from":             req.Limit * req.Page,
		"sort": []any{
			map[string]any{
				sortField: map[string]any{"order": sortDir},
			},
		},
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
		return SearchResponse{}, fmt.Errorf("failed search request %d", osResp.StatusCode)
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

// KeywordFacet is a single aggregation bucket from OpenSearch.
type KeywordFacet struct {
	Value string
	Count int64
}

// GetKeywordFacets returns distinct values and counts for any keyword (or keyword subfield) in the index.
// field should be a keyword field or a .keyword subfield (e.g. "gameSystem.keyword").
// size controls the maximum number of buckets returned.
func (r *EventRepo) GetKeywordFacets(ctx context.Context, field string, size int) ([]KeywordFacet, error) {
	body := map[string]any{
		"size": 0,
		"aggs": map[string]any{
			"facet_values": map[string]any{
				"terms": map[string]any{
					"field": field,
					"size":  size,
					"order": map[string]any{"_key": "asc"},
				},
			},
		},
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal facet request: %w", err)
	}

	osReq := opensearchapi.SearchRequest{
		Index: []string{r.eventIndex},
		Body:  bytes.NewReader(bodyBytes),
	}

	osResp, err := osReq.Do(ctx, r.client)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := osResp.Body.Close(); err != nil {
			r.logger.Err(err).Msg("failed to close facet response body")
		}
	}()

	if osResp.IsError() {
		r.logger.Error().Msgf("facet request failed. Raw response: %s", osResp.String())
		return nil, fmt.Errorf("failed facet request %d", osResp.StatusCode)
	}

	var raw struct {
		Aggregations struct {
			FacetValues struct {
				Buckets []struct {
					Key      string `json:"key"`
					DocCount int64  `json:"doc_count"`
				} `json:"buckets"`
			} `json:"facet_values"`
		} `json:"aggregations"`
	}

	buff := bytes.NewBuffer([]byte{})
	if _, err := buff.ReadFrom(osResp.Body); err != nil {
		return nil, fmt.Errorf("failed to read facet response body: %w", err)
	}
	if err := json.Unmarshal(buff.Bytes(), &raw); err != nil {
		return nil, fmt.Errorf("failed to unmarshal facet response: %w", err)
	}

	var facets []KeywordFacet
	for _, b := range raw.Aggregations.FacetValues.Buckets {
		if b.Key == "" {
			continue
		}
		facets = append(facets, KeywordFacet{Value: b.Key, Count: b.DocCount})
	}
	return facets, nil
}

func (r *EventRepo) FetchEvents(ctx context.Context, ids ...string) (FetchEventsResponse, error) {
	if len(ids) == 0 {
		return FetchEventsResponse{
			Found:   make(map[string]*Event),
			Missing: make(map[string]struct{}),
		}, nil
	}
	r.logger.Debug().Msgf("Fetching events for ids: %v", ids)
	fetchResp := FetchEventsResponse{}

	mgetBody := struct {
		IDs []string `json:"ids"`
	}{
		IDs: ids,
	}

	jsonBytes, err := json.Marshal(mgetBody)
	if err != nil {
		return fetchResp, fmt.Errorf("failed to convert the id list to a json request body: %w", err)
	}

	r.logger.Debug().Msgf("Raw mget request body: %s", string(jsonBytes))

	req := opensearchapi.MgetRequest{
		Index: r.eventIndex,
		Body:  bytes.NewReader(jsonBytes),
	}

	resp, err := req.Do(ctx, r.client)
	if err != nil {
		return fetchResp, err
	}

	defer func() {
		if resp.Body != nil {
			if err := resp.Body.Close(); err != nil {
				r.logger.Err(err).Msg("failed to clost mget response body")
			}
		}
	}()

	if resp.IsError() {
		r.logger.Error().Msgf("mget request failed. Raw response: %s", resp.String())
		return fetchResp, fmt.Errorf("failed fetch request %d", resp.StatusCode)
	}

	r.logger.Debug().Msgf("raw mget response body: %s", resp.String())

	var (
		response      eventMgetResponse
		foundEvents   = make(map[string]*Event)
		missingEvents = make(map[string]struct{})
		buff          = bytes.NewBuffer([]byte{})
	)

	if _, err := buff.ReadFrom(resp.Body); err != nil {
		return fetchResp, fmt.Errorf("failed to read search response body: %w", err)
	}

	if err := json.Unmarshal(buff.Bytes(), &response); err != nil {
		return fetchResp, fmt.Errorf("failed to unmarshal mget response: %w", err)
	}

	if response.Error != nil {
		return fetchResp, fmt.Errorf("os error on mget request: %s", response.Error.Reason)
	}

	for _, d := range response.Docs {
		if d.Found {
			foundEvents[d.ID] = d.Event
		} else {
			missingEvents[d.ID] = struct{}{}
		}
	}

	return FetchEventsResponse{
		Found:   foundEvents,
		Missing: missingEvents,
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

type bulkUpdateAction struct {
	Doc *Event `json:"doc"`
}

type bulkResponse struct {
	Took   int64 `json:"took"`
	Errors bool  `json:"errors"`
	// maps bulk action to response
	// ie "create": {}
	Items []map[string]bulkActionResponse `json:"items"`
}

type bulkActionResponse struct {
	Index          string `json:"_index"`
	ID             string `json:"_id"`
	Version        int64  `json:"_version"`
	SequenceNumber int64  `json:"_seq_no"`
	PrimaryTerm    int64  `json:"_primary_term"`
	Result         string `json:"result"`
	Shards         struct {
		Total      int64 `json:"total"`
		Successful int64 `json:"successful"`
		Failed     int64 `json:"failed"`
	} `json:"_shards"`
	Status int      `json:"status"`
	Error  *osError `json:"error,omitempty"`
}

type eventMgetResponse struct {
	Docs []struct {
		Index          string `json:"_index"`
		ID             string `json:"_id"`
		Version        int64  `json:"_version"`
		SequenceNumber int64  `json:"_seq_no"`
		PrimaryTerm    int64  `json:"_primary_term"`
		Found          bool   `json:"found"`
		Event          *Event `json:"_source,omitempty"`
	} `json:"docs"`
	Error *osError `json:"error,omitempty"`
}

type osError struct {
	Type      string `json:"type"`
	Reason    string `json:"reason"`
	Index     string `json:"index"`
	IndexUUID string `json:"index_uuid"`
}
