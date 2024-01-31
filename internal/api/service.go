package api

import (
	"context"
	"errors"

	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/rs/zerolog"
)

type GenconBuddyAPI struct {
	logger       *zerolog.Logger
	eventHandler *EventHandler
	server       *http.Server
}

func NewGenconBuddyAPI(logger *zerolog.Logger) *GenconBuddyAPI {

	gcb := &GenconBuddyAPI{
		logger: logger,
	}

	logger.Info().Msg("Initializing GenconBuddyAPI")

	logger.Info().Msg("Initializing EventHandler")
	eventHandler := NewEventHandler(logger)
	eventHandler.Register()
	logger.Info().Msg("Finidhsed initializing EventHandler")

	logger.Info().Msg("Initializing HTTP Server")
	gcb.server = &http.Server{
		Addr:    ":8080",
		Handler: restful.DefaultContainer,
	}
	logger.Info().Msg("Finished initializing HTTP Server")

	gcb.eventHandler = eventHandler
	logger.Info().Msg("Finished initializing GenconBuddyAPI")

	return gcb
}

// Start starts the GennconBuddyAPI asyncronously
func (gb *GenconBuddyAPI) Start() {
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
