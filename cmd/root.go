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
	"github.com/spf13/viper"
)

const (
	flagVerbosity    = "verbosity"
	flagBatchSize    = "batch_size"
	flagOSAddress    = "os_address"
	flagOSUsername   = "os_username"
	flagOSPassword   = "os_password"
	flagOSEventIndex = "event_index"
)

var (
	gcbRootCmd = &cobra.Command{
		Use:   "gcb",
		Short: "GenConBuddy is the cli helper for initiating, setting up, and maintaining the GenConBuddy API Service.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var err error
			verbosity := viper.GetString(flagVerbosity)
			logVerbosity := zerolog.InfoLevel
			if verbosity != "" {
				logVerbosity, err = zerolog.ParseLevel(verbosity)
				if err != nil {
					return err
				}
			}

			config := app.AppConfig{
				OSAddress:  viper.GetString(flagOSAddress),
				OSUsername: viper.GetString(flagOSUsername),
				OSPassword: viper.GetString(flagOSPassword),
				EventIndex: viper.GetString(flagOSEventIndex),
				BatchSize:  viper.GetInt(flagBatchSize),
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
	viper.AutomaticEnv()
	// viper.New().SetDefault(flagBatchSize, 100)

	gcbRootCmd.PersistentFlags().StringP(flagVerbosity, "v", "info", "set the log verbosity.")
	viper.BindPFlag("VERBOSITY", gcbRootCmd.PersistentFlags().Lookup(flagVerbosity))

	gcbRootCmd.PersistentFlags().String(flagOSAddress, "", "the address to connect to with opensearch.")
	viper.BindPFlag("OS_ADDRESS", gcbRootCmd.PersistentFlags().Lookup(flagOSAddress))

	gcbRootCmd.PersistentFlags().String(flagOSUsername, "admin", "the username to connect to the cluster with. Defaults to admin.")
	viper.BindPFlag("OS_USERNAME", gcbRootCmd.PersistentFlags().Lookup(flagOSUsername))

	gcbRootCmd.PersistentFlags().String(flagOSPassword, "", "the password for the user connecting to opensearch.")
	viper.BindPFlag("OS_PASSWORD", gcbRootCmd.PersistentFlags().Lookup(flagOSPassword))

	gcbRootCmd.PersistentFlags().String(flagOSEventIndex, "event_index", "Root index name. This value is used as the primary event index. Defaults to 'event_index'")
	viper.BindPFlag("EVENT_INDEX", gcbRootCmd.PersistentFlags().Lookup(flagOSEventIndex))

	gcbRootCmd.PersistentFlags().Int(flagBatchSize, 100, "Size of batches/pages for interactin with opensearch.")
	viper.BindPFlag("BATCH_SIZE", gcbRootCmd.PersistentFlags().Lookup(flagBatchSize))

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
