package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/gencon_buddy_api/cmd/api"
	"github.com/gencon_buddy_api/cmd/app"
	"github.com/gencon_buddy_api/cmd/data"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var (
	verbosity string
	config    app.AppConfig

	gcbRootCmd = &cobra.Command{
		Use:   "gcb",
		Short: "GenConBuddy is the cli helper for initiating, setting up, and maintaining the GenConBuddy API Service.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			logVerbosity := zerolog.InfoLevel
			var err error
			if verbosity != "" {
				logVerbosity, err = zerolog.ParseLevel(verbosity)
				if err != nil {
					return err
				}
			}

			logger := zerolog.New(
				zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339},
			).Level(logVerbosity).With().Timestamp().Caller().Logger()

			gcbApp, err := app.NewApp(logger, config)
			if err != nil {
				return err
			}

			cmd.SetContext(app.ContextWithApp(cmd.Context(), gcbApp))

			return nil
		},
	}
)

func init() {
	gcbRootCmd.PersistentFlags().StringVarP(&verbosity, "verbosity", "v", "info", "set the log verbosity.")
	gcbRootCmd.PersistentFlags().StringVar(&config.OSAddress, "osAddress", "", "the address to connect to with opensearch.")
	gcbRootCmd.PersistentFlags().StringVar(&config.OSUsername, "osUsername", "admin", "the username to connect to the cluster with. Defaults to admin.")
	gcbRootCmd.PersistentFlags().StringVar(&config.OSPassword, "osPassword", "", "the password for the user connecting to opensearch.")
	gcbRootCmd.PersistentFlags().StringVar(&config.EventIndex, "eventIndex", "event_index", "Root index name. This value is used as the primary event index. Defaults to 'event_index'")
	gcbRootCmd.PersistentFlags().IntVar(&config.BatchSize, "batchSize", 100, "Size of batches/pages for interactin with opensearch.")

	gcbRootCmd.AddCommand(api.ServiceCmd)
	gcbRootCmd.AddCommand(data.Cmd)
}

func Execute() {
	if err := gcbRootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// init()? https://github.com/spf13/cobra/blob/main/site/content/user_guide.md

// initConfig() with viper?
