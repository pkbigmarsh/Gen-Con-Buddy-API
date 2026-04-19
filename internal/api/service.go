package api

import (
	"context"
	"errors"
	"fmt"

	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/gencon_buddy_api/internal/event"
	"github.com/rs/cors"
	"github.com/rs/zerolog"
)

type GenconBuddyAPI struct {
	logger           *zerolog.Logger
	eventHandler     *EventHandler
	changeLogHandler *ChangeLogHandler
	server           *http.Server
	eventRepo        *event.EventRepo
}

func NewGenconBuddyAPI(logger *zerolog.Logger, eventRepo *event.EventRepo, port int) *GenconBuddyAPI {

	gcb := &GenconBuddyAPI{
		logger: logger,
	}

	logger.Info().Msg("Initializing GenconBuddyAPI")

	logger.Info().Msg("Initializing EventHandler")
	eventHandler := NewEventHandler(logger, NewEventManager(logger, eventRepo))
	eventHandler.Register()
	gcb.eventHandler = eventHandler
	logger.Info().Msg("Finidhsed initializing EventHandler")

	logger.Info().Msg("Initializing ChangLogHandler")
	changeLogHandler := NewChangeLogHandler(logger)
	if err := changeLogHandler.Register(); err != nil {
		logger.Err(err).Msg("Failed to create the ChangeLogHandler successfully")
	}
	gcb.changeLogHandler = changeLogHandler
	logger.Info().Msg("Finished initializing ChangeLogHandler")

	logger.Info().Msg("Initializing HTTP Server")
	logger.Debug().Msgf("Listening to port %d", port)
	gcb.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: cors.AllowAll().Handler(restful.DefaultContainer),
	}
	logger.Info().Msg("Finished initializing HTTP Server")

	logger.Info().Msg("Finished initializing GenconBuddyAPI")

	gcb.eventRepo = eventRepo

	return gcb
}

// Start starts the GennconBuddyAPI asyncronously
func (gb *GenconBuddyAPI) Start() {
	gb.logger.Info().Msg("Starting http server")

	go func() {
		err := gb.server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			gb.logger.Error().Err(err).Msg("Failed to close the server gracefully")
		}
	}()
}

// Stop attempts to stop the GenconBuddyAPI and returns an error with any issues
func (gb *GenconBuddyAPI) Stop(ctx context.Context) error {
	return gb.server.Shutdown(ctx)
}
