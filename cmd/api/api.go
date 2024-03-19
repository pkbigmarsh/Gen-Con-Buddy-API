package api

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gencon_buddy_api/cmd/app"
	"github.com/gencon_buddy_api/internal/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	flagPort = "port"
)

var (
	ServiceCmd = &cobra.Command{
		Use:   "api",
		Short: "Starts the GCB API Service",
		Long:  "Runs the api service as a blocking command.",
		RunE:  run,
	}
)

func init() {
	ServiceCmd.Flags().IntP(flagPort, "p", 8080, "The port for the api service to listen to")
	viper.BindPFlag("PORT", ServiceCmd.Flags().Lookup(flagPort))
}

func run(cmd *cobra.Command, _ []string) error {
	gcb := app.GetAppFromContext(cmd.Context())
	if gcb == nil {
		return fmt.Errorf("failed to get GCB App from context when starting api service")
	}

	port := viper.GetInt(flagPort)

	mainCtx, mainCancel := context.WithCancel(context.Background())
	apiService := api.NewGenconBuddyAPI(&gcb.Logger, gcb.EventRepo, port)

	gracefullShutdown := make(chan os.Signal, 1)
	signal.Notify(gracefullShutdown, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-gracefullShutdown
		gcb.Logger.Info().Msg("Recieved Shutdown signal...")
		apiService.Stop(mainCtx)
		mainCancel()
	}()

	apiService.Start()

	for range mainCtx.Done() {
		return nil
	}

	return nil
}
