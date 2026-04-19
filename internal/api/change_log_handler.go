package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/emicklei/go-restful/v3"
	"github.com/gencon_buddy_api/gcbapi"
	"github.com/rs/zerolog"
)

// ChangeLogHandler is the API handler for all /api/changelog/* endpoints
type ChangeLogHandler struct {
	logger *zerolog.Logger
	ws     *restful.WebService
	// manager ChangeLogManager
}

// NewChangeLogHandler instantiates a [ChangeLogHandler]
func NewChangeLogHandler(logger *zerolog.Logger) *ChangeLogHandler {
	return &ChangeLogHandler{
		logger: logger,
		ws:     new(restful.WebService),
		// manager: manager
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

	c.logger.Debug().Msgf("mocking list change log with limit [%d]", limit)
	response = gcbapi.ListChangeLogsResponse{
		Entries: []gcbapi.ChangeLogSummary{
			{
				ID:           "example",
				Date:         time.Now().UTC().Format(time.RFC3339),
				UpdatedCount: 1,
				CreatedCount: 10,
				DeletedCount: 124,
			},
		},
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

	c.logger.Debug().Msgf("mocking fetch change log with id [%s]", id)
	response = gcbapi.FetchChangeLogResponse{
		Entry: gcbapi.ChangeLogEntry{
			ID:   id,
			Date: time.Now().UTC().Format(time.RFC3339),
			UpdatedEvents: []gcbapi.Event{
				{
					ID:   "ENT24ND246100",
					Type: "event",
					Attributes: gcbapi.EventAttributes{
						GameID:                   "ENT24ND246100",
						Year:                     0,
						Group:                    "Unprepared Casters",
						Title:                    "Unprepared Casters Live!",
						ShortDescription:         "Join Haley Whipjack, Amelia Som, and some of your other favorite Unprepared Casters guests for the show's first ever live event!",
						LongDescription:          "",
						EventType:                "ENT - Entertainment Events",
						GameSystem:               "",
						RulesEdition:             "",
						MinPlayers:               20,
						MaxPlayers:               100,
						AgeRequired:              "Teen (13+)",
						ExperienceRequired:       "None (You've never played before - rules will be taught)",
						MaterialsProvided:        "",
						MaterialsRequired:        "No",
						MaterialsRequiredDetails: "",
						StartDateTime:            time.Now().UTC(),
						Duration:                 2,
						EndDateTime:              time.Now().UTC(),
						GMNames:                  "Haley Meyer",
						Website:                  "",
						Email:                    "",
						Tournament:               "No",
						RoundNumber:              1,
						TotalRounds:              1,
						MinimumPlayTime:          0,
						AttendeeRegistration:     "Yes, they can register for this round without having played in any other events",
						Cost:                     4,
						Location:                 "Union Station",
						RoomName:                 "Grand Hall",
						TableNumber:              "",
						SpecialCategory:          "none",
						TicketsAvailableTime:     54,
						LastModified:             time.Now().UTC(),
						AlsoRuns:                 time.Now().UTC(),
						Prize:                    "",
						RulesComplexity:          "",
						OriginalOrder:            0,
					},
				},
			},
			DeletedEvents: []gcbapi.Event{
				{
					ID:   "SEM24ND246101",
					Type: "event",
					Attributes: gcbapi.EventAttributes{
						GameID:                   "SEM24ND246101",
						Year:                     0,
						Group:                    "Critical Chaos Entertainment",
						Title:                    "DM Do's and Don'ts Q&A",
						ShortDescription:         "Join us as we talk about how to DM! Learn world building, safety tools, and have your DM/GM Questions answered! ",
						LongDescription:          "Join Offbeat_Outlaw, Ben Brainard, Sphinx (CircleDM), Endevourance and SirFeffers as they talk about how to DM, world building, safety tools, and answer your learning to DM/GM questions! ",
						EventType:                "SEM - Seminar",
						GameSystem:               "",
						RulesEdition:             "",
						MinPlayers:               50,
						MaxPlayers:               70,
						AgeRequired:              "Everyone (6+)",
						ExperienceRequired:       "None (You've never played before - rules will be taught)",
						MaterialsProvided:        "",
						MaterialsRequired:        "No",
						MaterialsRequiredDetails: "",
						StartDateTime:            time.Now().UTC(),
						Duration:                 2,
						EndDateTime:              time.Now().UTC(),
						GMNames:                  "Jeff Samuels",
						Website:                  "",
						Email:                    "",
						Tournament:               "No",
						RoundNumber:              1,
						TotalRounds:              1,
						MinimumPlayTime:          0,
						AttendeeRegistration:     "Yes, they can register for this round without having played in any other events",
						Cost:                     0,
						Location:                 "Stadium",
						RoomName:                 "Meeting Room 12",
						TableNumber:              "",
						SpecialCategory:          "none",
						TicketsAvailableTime:     48,
						LastModified:             time.Now().UTC(),
						AlsoRuns:                 time.Now().UTC(),
						Prize:                    "",
						RulesComplexity:          "",
						OriginalOrder:            0,
					},
				},
			},
			CreatedEvents: []gcbapi.Event{
				{
					ID:   "SPA24ND246103",
					Type: "event",
					Attributes: gcbapi.EventAttributes{
						GameID:                   "SPA24ND246103",
						Year:                     0,
						Group:                    "The Revel Alliance",
						Title:                    "Court Dancing: +1 Performance, +1 Diplomacy",
						ShortDescription:         "In your eternal quest for just one more point of diplomacy at formal elven functions, why not explore dance? All new dances for 2024!",
						LongDescription:          "",
						EventType:                "SPA - Supplemental Activities",
						GameSystem:               "",
						RulesEdition:             "",
						MinPlayers:               8,
						MaxPlayers:               80,
						AgeRequired:              "Teen (13+)",
						ExperienceRequired:       "None (You've never played before - rules will be taught)",
						MaterialsProvided:        "",
						MaterialsRequired:        "No",
						MaterialsRequiredDetails: "",
						StartDateTime:            time.Now().UTC(),
						Duration:                 1,
						EndDateTime:              time.Now().UTC(),
						GMNames:                  "Whitney Rowlett",
						Website:                  "moonlightdance.co",
						Email:                    "whitney@moonlightdance.co",
						Tournament:               "No",
						RoundNumber:              1,
						TotalRounds:              1,
						MinimumPlayTime:          0,
						AttendeeRegistration:     "Yes, they can register for this round without having played in any other events",
						Cost:                     12,
						Location:                 "Westin",
						RoomName:                 "House",
						TableNumber:              "",
						SpecialCategory:          "none",
						TicketsAvailableTime:     39,
						LastModified:             time.Now().UTC(),
						AlsoRuns:                 time.Now().UTC(),
						Prize:                    "",
						RulesComplexity:          "",
						OriginalOrder:            0,
					},
				},
			},
		},
		Error: "",
	}
}
