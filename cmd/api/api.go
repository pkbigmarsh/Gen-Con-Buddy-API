package api

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gencon_buddy_api/internal/api"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var ServiceCmd = &cobra.Command{
	Use:   "api",
	Short: "Starts the GCB API Service",
	Long:  "Runs the api service as a blocking command.",
	Run:   run,
}

func run(_ *cobra.Command, _ []string) {
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
