package api

import (
	"github.com/emicklei/go-restful/v3"
	"github.com/rs/zerolog"
)

// EventHandler is the API Handler for all /events/* endpoints.
type EventHandler struct {
	logger *zerolog.Logger
	ws     *restful.WebService
}

// NewEventHandler instantiates an [EventHandler].
func NewEventHandler(logger *zerolog.Logger) *EventHandler {
	return &EventHandler{
		logger: logger,
		ws:     new(restful.WebService),
	}
}

// Register registers all event endpoints with the restful service.
func (e *EventHandler) Register() {
	e.ws.Path("/events")
	e.ws.Consumes(restful.MIME_JSON)
	e.ws.Produces(restful.MIME_JSON)

	e.ws.Route(e.ws.GET("/search").To(e.Search).
		Doc("Search for events").
		Writes(struct{}{}). // TODO
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

	resp.Write([]byte("Success"))
}
