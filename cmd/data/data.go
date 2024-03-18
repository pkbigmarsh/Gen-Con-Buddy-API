package data

import (
	"fmt"

	"github.com/gencon_buddy_api/cmd/app"
	"github.com/gencon_buddy_api/cmd/data/initialize"
	"github.com/spf13/cobra"
)

var (
	eventIndex string

	Cmd = &cobra.Command{
		Use:   "data",
		Short: "Data commands to setup, load, and update the opensearch cluster",
		Long:  "If not sub commands are provided, then all will be executed sequentially",
		RunE:  run,
	}
)

func init() {
	Cmd.PersistentFlags().BoolP("clean", "c", false, "cleans all indicies before initilizing the data")
	Cmd.PersistentFlags().StringP("filepath", "f", "", "the filepath of the csv event data to load")

	Cmd.AddCommand(initialize.InitCmd)
}

func run(cmd *cobra.Command, args []string) error {
	gcb := app.GetAppFromContext(cmd.Context())
	if gcb == nil {
		return fmt.Errorf("couldn't initialize gcb app context")
	}

	gcb.Logger.Debug().Msgf("Executing data command with args: eventIndex=%s", eventIndex)

	if err := initialize.InitCmd.RunE(cmd, args); err != nil {
		return fmt.Errorf("failed to intialize the data: %w", err)
	}

	return nil
}
