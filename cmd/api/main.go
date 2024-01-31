package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gencon_buddy_api/internal/api"
	"github.com/rs/zerolog"
)

func main() {
	logger := zerolog.New(
		zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339},
	).Level(zerolog.TraceLevel).With().Timestamp().Caller().Logger()
	mainCtx, mainCancel := context.WithCancel(context.Background())
	apiService := api.NewGenconBuddyAPI(&logger)

	gracefullShutdown := make(chan os.Signal, 1)
	signal.Notify(gracefullShutdown, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-gracefullShutdown
		logger.Info().Msg("Recieved Shutdown signal...")
		apiService.Stop(mainCtx)
		mainCancel()
	}()

	apiService.Start()

	for range mainCtx.Done() {
		return
	}
}
