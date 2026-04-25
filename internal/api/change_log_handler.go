package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/emicklei/go-restful/v3"
	"github.com/gencon_buddy_api/gcbapi"
	"github.com/rs/zerolog"
)

// ChangeLogHandler is the API handler for all /api/changelog/* endpoints
type ChangeLogHandler struct {
	logger  *zerolog.Logger
	ws      *restful.WebService
	manager ChangeLogManager
}

// NewChangeLogHandler instantiates a [ChangeLogHandler]
func NewChangeLogHandler(logger *zerolog.Logger, manager ChangeLogManager) *ChangeLogHandler {
	return &ChangeLogHandler{
		logger:  logger,
		ws:      new(restful.WebService),
		manager: manager,
	}
}

// Register all change log endpoints with the restful service
func (c *ChangeLogHandler) Register() error {
	if c == nil {
		return fmt.Errorf("cannot register the changelog endpoints with no ChangeLogHandler")
	}

	if c.ws == nil {
		return fmt.Errorf("canot register the changelog endpoints with no restful.WebService")
	}

	c.ws.Path("/api/changelog")
	c.ws.Consumes(restful.MIME_JSON)
	c.ws.Produces(restful.MIME_JSON)

	c.ws.Route(c.ws.GET("/list").To(c.ListChangeLogs).
		Doc("List the Change Logs as summaries. Only the event modification counts will be shown.").
		Writes(gcbapi.ListChangeLogsResponse{}).
		Param(c.ws.QueryParameter("limit", "The number of change log entries to return. Default is 6").
			DataType("int").DefaultValue("6").Minimum(0).Maximum(100)))

	c.ws.Route(c.ws.GET("/fetch").To(c.FetchChangeLog).
		Doc("Fetch the desired change log, fully hydrating event details").
		Writes(gcbapi.FetchChangeLogResponse{}).
		Param(c.ws.QueryParameter("id", "What change log id to fetch").
			DataType("string").Required(true)))

	restful.Add(c.ws)

	return nil
}

// ListChangeLogs list as many change log summaries as desired
func (c *ChangeLogHandler) ListChangeLogs(req *restful.Request, resp *restful.Response) {
	var (
		response gcbapi.ListChangeLogsResponse
		limit    int = 6
	)

	defer func() {
		responseBody, err := json.Marshal(response)
		if err != nil {
			c.logger.Err(err).Msg("failed to marshal list change log response")
			resp.WriteErrorString(http.StatusInternalServerError, "failed to write response")
			return
		}

		_, err = resp.Write(responseBody)
		if err != nil {
			c.logger.Err(err).Msg("failed to write rest response")
			resp.WriteErrorString(http.StatusInternalServerError, "failed to write response")
			return
		}
	}()

	for queryParam, values := range req.Request.URL.Query() {
		switch queryParam {
		case "limit":
			if len(values) > 1 {
				resp.WriteHeader(http.StatusBadRequest)
				response.Error = "only 1 limit query parameter is allowed"
				return
			}

			i, err := strconv.Atoi(values[0])
			if err != nil {
				resp.WriteHeader(http.StatusBadRequest)
				response.Error = fmt.Sprintf("invalid integer for limti: %w", err)
				return
			}

			limit = i
		default:
			c.logger.Warn().Msgf("list change log entries attempted with unknown query parameter [%s]", queryParam)
			resp.WriteHeader(http.StatusBadRequest)
			response.Error = fmt.Sprintf("unsupported query paramter supplied [%s]", queryParam)
			return
		}
	}

	summaries, err := c.manager.ListChangeLogSummaries(req.Request.Context(), limit)
	if err != nil {
		c.logger.Err(err).Msg("failed to list change log summaries")
		resp.WriteHeader(http.StatusInternalServerError)
		response.Error = "failed to list change log summaries"
		return
	}

	response = gcbapi.ListChangeLogsResponse{
		Entries: summaries,
	}
	resp.WriteHeader(http.StatusOK)
}

// FetchChangeLog fetches the specific changelog based on the id
func (c *ChangeLogHandler) FetchChangeLog(req *restful.Request, resp *restful.Response) {
	var (
		response gcbapi.FetchChangeLogResponse
		id       string
	)

	defer func() {
		responseBody, err := json.Marshal(response)
		if err != nil {
			c.logger.Err(err).Msg("failed to marshal fetch change log response")
			resp.WriteErrorString(http.StatusInternalServerError, "failed to write response")
			return
		}

		_, err = resp.Write(responseBody)
		if err != nil {
			c.logger.Err(err).Msg("failed to write rest response")
			resp.WriteErrorString(http.StatusInternalServerError, "failed to write response")
			return
		}
	}()

	for queryParam, values := range req.Request.URL.Query() {
		switch queryParam {
		case "id":
			if len(values) > 1 {
				resp.WriteHeader(http.StatusBadRequest)
				response.Error = "only 1 id query parameter is allowed"
				return
			}

			id = values[0]
		default:
			c.logger.Warn().Msgf("fetch change log entries attempted with unknown query parameter [%s]", queryParam)
			resp.WriteHeader(http.StatusBadRequest)
			response.Error = fmt.Sprintf("unsupported query paramter supplied [%s]", queryParam)
			return
		}
	}
	if id == "" {
		resp.WriteHeader(http.StatusBadRequest)
		response.Error = "fetch change log entry requires an id query param"
		return
	}

	entry, err := c.manager.FetchChangeLogEntry(req.Request.Context(), id)
	if err != nil {
		c.logger.Err(err).
			Str("change_log_id", id).
			Msg("failed to fetch change log")
		resp.WriteHeader(http.StatusInternalServerError)
		response.Error = "failed to fetch change log"
		return
	}

	c.logger.Debug().Msgf("mocking fetch change log with id [%s]", id)
	response = gcbapi.FetchChangeLogResponse{
		Entry: entry,
	}
}
