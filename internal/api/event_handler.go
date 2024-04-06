package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"github.com/gencon_buddy_api/gcbapi"
	"github.com/gencon_buddy_api/internal/event"
	"github.com/rs/zerolog"
)

// EventHandler is the API Handler for all /events/* endpoints.
type EventHandler struct {
	logger  *zerolog.Logger
	ws      *restful.WebService
	manager EventManager
}

// NewEventHandler instantiates an [EventHandler].
func NewEventHandler(logger *zerolog.Logger, manager EventManager) *EventHandler {
	return &EventHandler{
		logger:  logger,
		ws:      new(restful.WebService),
		manager: manager,
	}
}

// Register registers all event endpoints with the restful service.
func (e *EventHandler) Register() {
	e.ws.Path("/api/events")
	e.ws.Consumes(restful.MIME_JSON)
	e.ws.Produces("application/vnd.api+json")

	e.ws.Route(e.ws.GET("/search").To(e.Search).
		Doc("Search for events").
		Writes(gcbapi.EventSearchResponse{}). // TODO
		Param(e.ws.QueryParameter("filter", "The value to perform the search with.").
			DataType("string").AllowEmptyValue(true)).
		Param(e.ws.QueryParameter("limit", "The number of events to return. Default is 100.").
			DataType("int").DefaultValue("100").Minimum(0).Maximum(5000)).
		Param(e.ws.QueryParameter("page", "What page of events to return. Pages are based on the limit. Default is 0").
			DataType("int").DefaultValue("0").Minimum(0).Maximum(100)).
		Param(e.ws.QueryParameter("sort", "What field to sert the events on formatted by {field name}.{asc|desc}. Can sort by any field in the event.").
			DataType("string").DefaultValue("")))

	restful.Add(e.ws)
}

// Search handles /events/search api calls
func (e *EventHandler) Search(req *restful.Request, resp *restful.Response) {
	var (
		// sort     string // TODO
		response  gcbapi.EventSearchResponse
		searchReq = event.SearchRequest{
			Page:  0,
			Limit: 100,
		}
	)

	defer func() {
		responseBody, err := json.Marshal(response)
		if err != nil {
			e.logger.Err(err).Msg("failed to marshal event search response")
			resp.WriteErrorString(http.StatusInternalServerError, "failed to write response")
			return
		}

		_, err = resp.Write(responseBody)
		if err != nil {
			e.logger.Err(err).Msg("failed to write response by")
			resp.WriteErrorString(http.StatusInternalServerError, "failed to write response")
			return
		}
	}()

	for queryParam, values := range req.Request.URL.Query() {
		switch queryParam {
		case "limit":
			if len(values) > 1 {
				resp.WriteHeader(http.StatusBadRequest)
				response.Error = &gcbapi.Error{
					Status: "bad request",
					Detail: "only 1 limit query parameter is allowed",
				}
				return
			}

			i, err := strconv.Atoi(values[0])
			if err != nil {
				resp.WriteHeader(http.StatusBadRequest)
				response.Error = &gcbapi.Error{
					Status: "bad request",
					Detail: fmt.Errorf("invalid integer for limti: %w", err).Error(),
				}
				return
			}

			searchReq.Limit = i
		case "page":
			if len(values) > 1 {
				resp.WriteHeader(http.StatusBadRequest)
				response.Error = &gcbapi.Error{
					Status: "bad request",
					Detail: "only 1 page query parameter is allowed",
				}
				return
			}

			i, err := strconv.Atoi(values[0])
			if err != nil {
				resp.WriteHeader(http.StatusBadRequest)
				response.Error = &gcbapi.Error{
					Status: "bad request",
					Detail: fmt.Errorf("invalid integer for page: %w", err).Error(),
				}
				return
			}

			searchReq.Page = i
		case "sort":
			// TODO lol
		default:
			// search term?
			searchTerm, err := event.NewSearchField(queryParam, strings.Join(values, ","))
			if err != nil {
				resp.WriteHeader(http.StatusBadRequest)
				response.Error = &gcbapi.Error{
					Status: "bad request",
					Detail: fmt.Errorf("invalid search query param %s: %w", queryParam, err).Error(),
				}
				return
			}

			e.logger.Debug().Msgf("parsed search term from query param %s and values %v: %+v", queryParam, values, searchTerm)

			searchReq.Terms = append(searchReq.Terms, searchTerm)
		}
	}

	var err error

	response.Meta.Total, response.Data, err = e.manager.Search(req.Request.Context(), searchReq)
	if err != nil {
		e.logger.Err(err).Msgf("Failed to perform search request [%v]", searchReq)
		resp.WriteHeader(http.StatusInternalServerError)
		response.Error = &gcbapi.Error{
			Status: "internal server error",
			Detail: "failed executing search request",
		}
	}

	resp.WriteHeader(http.StatusOK)
}
