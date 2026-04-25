package changelog

import (
	"bytes"
	"context"
	"encoding/json"
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

// Repo controls talking to the OpenSearch cluster for change log details
type Repo struct {
	logger         *zerolog.Logger
	client         *opensearch.Client
	batchSize      int
	changeLogIndex string
}

// NewRepo instantiates a new Repo
func NewRepo(logger *zerolog.Logger, client *opensearch.Client, batchSize int, changeLogIndex string) *Repo {
	return &Repo{
		logger:         logger,
		client:         client,
		batchSize:      batchSize,
		changeLogIndex: changeLogIndex,
	}
}

func (r *Repo) CreateEntries(ctx context.Context, entries ...*Entry) ([]error, error) {
	return r.WriteEntries(ctx, createAction, entries)
}

func (r *Repo) WriteEntries(ctx context.Context, action string, entries []*Entry) ([]error, error) {
	r.logger.Info().Msgf("Writing %d entries, expected batch count: %d", len(entries), len(entries)/r.batchSize)
	var (
		body      = ""
		batchSize = 0
		docErrs   []error
	)
	for _, e := range entries {
		body += fmt.Sprintf(bulkMeta, action, r.changeLogIndex, e.ID) + "\n"
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
			docErrs = append(docErrs, fmt.Errorf("failed to marshal change log entry %s: %w", e.ID, err))
			continue
		}

		body += string(docJson) + "\n"
		batchSize++

		if batchSize >= r.batchSize {
			errs, err := r.writeEntries(ctx, action, body)
			docErrs = append(docErrs, errs...)
			if err != nil {
				return docErrs, fmt.Errorf("failed to execute change log write batch: %s", err)
			}

			body = ""
			batchSize = 0
		}
	}

	if batchSize > 0 {
		errs, err := r.writeEntries(ctx, action, body)
		docErrs = append(docErrs, errs...)
		return docErrs, err
	}

	return docErrs, nil
}

func (r *Repo) writeEntries(ctx context.Context, action string, body string) ([]error, error) {
	r.logger.Debug().Msgf("Writing entries with request: %s", body)
	bulkWriteReq := opensearchapi.BulkRequest{
		Index: r.changeLogIndex,
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

	r.logger.Debug().Msgf("Raw write entries response: %s", resp.String())

	if resp.IsError() {
		return nil, fmt.Errorf("failted to write some entries, code [%d] headers [%v]", resp.StatusCode, resp.Header)
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
		r.logger.Debug().Msg("Succesfully short circuiting")
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

func (r *Repo) UpdateEntries(ctx context.Context, entries []*Entry) ([]error, error) {
	return r.WriteEntries(ctx, updateAction, entries)
}

func (r *Repo) List(ctx context.Context, req ListEntriesRequest) ([]*Entry, error) {
	r.logger.Debug().Msgf("performing search request: %+v", req)
	if req.Limit <= 0 {
		return nil, fmt.Errorf("limit cannot be less than 1, got %d", req.Limit)
	}

	sortField := "date"
	sortDir := "desc"

	searchBody := map[string]any{
		"track_total_hits": true,
		"size":             req.Limit,
		"sort": []any{
			map[string]any{
				sortField: map[string]any{"order": sortDir},
			},
		},
	}

	bodyBytes, err := json.Marshal(searchBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search request: %w", err)
	}

	r.logger.Debug().Msgf("Performing search request: %s", bodyBytes)

	osReq := opensearchapi.SearchRequest{
		Index: []string{r.changeLogIndex},
		Body:  bytes.NewReader(bodyBytes),
	}

	osResp, err := osReq.Do(ctx, r.client)
	if err != nil {
		return nil, err
	}
	defer func() {
		err := osResp.Body.Close()
		if err != nil {
			r.logger.Err(err).Msgf("failed to close search response body")
		}
	}()

	if osResp.IsError() {
		r.logger.Error().Msgf("search request failed. Raw response: %s", osResp.String())
		return nil, fmt.Errorf("failed search request %d", osResp.StatusCode)
	}

	r.logger.Debug().Msgf("search request debuging: %s", osResp.String())

	var (
		response entriesearchResponse
		entries  []*Entry
		buff     = bytes.NewBuffer([]byte{})
	)

	if _, err := buff.ReadFrom(osResp.Body); err != nil {
		return nil, fmt.Errorf("failed to read search response body: %w", err)
	}

	if err := json.Unmarshal(buff.Bytes(), &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal search response: %w", err)
	}

	for _, e := range response.Hits.Hits {
		entries = append(entries, e.Entry)
	}

	return entries, nil
}

func (r *Repo) FetchEntries(ctx context.Context, ids ...string) (FetchEntriesResponse, error) {
	r.logger.Debug().Msgf("Fetching entries for ids: %v", ids)
	fetchResp := FetchEntriesResponse{}

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
		Index: r.changeLogIndex,
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
		response       entryMgetResponse
		foundEntries   = make(map[string]*Entry)
		missingEntries = make(map[string]struct{})
		buff           = bytes.NewBuffer([]byte{})
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
			foundEntries[d.ID] = d.Event
		} else {
			missingEntries[d.ID] = struct{}{}
		}
	}

	return FetchEntriesResponse{
		Found:   foundEntries,
		Missing: missingEntries,
	}, nil
}

type entriesearchResponse struct {
	Hits struct {
		Total struct {
			Value int64 `json:"value"`
		} `json:"total"`
		MaxScore float64 `json:"max_score"`
		Hits     []struct {
			Index string  `json:"_index"`
			ID    string  `json:"_id"`
			Score float64 `json:"_score"`
			Entry *Entry  `json:"_source,omitempty"`
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
	Doc *Entry `json:"doc"`
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

type entryMgetResponse struct {
	Docs []struct {
		Index          string `json:"_index"`
		ID             string `json:"_id"`
		Version        int64  `json:"_version"`
		SequenceNumber int64  `json:"_seq_no"`
		PrimaryTerm    int64  `json:"_primary_term"`
		Found          bool   `json:"found"`
		Event          *Entry `json:"_source,omitempty"`
	} `json:"docs"`
	Error *osError `json:"error,omitempty"`
}

type osError struct {
	Type      string `json:"type"`
	Reason    string `json:"reason"`
	Index     string `json:"index"`
	IndexUUID string `json:"index_uuid"`
}
