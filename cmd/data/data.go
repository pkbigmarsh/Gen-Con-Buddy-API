package data

import (
	"github.com/gencon_buddy_api/cmd/data/initialize"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "data",
	Short: "Data commands to setup, load, and update the opensearch cluster",
	Long:  "If not sub commands are provided, then all will be executed sequentially",
	RunE:  run,
}

func run(cmd *cobra.Command, args []string) error {

	return nil
}

func init() {
	Cmd.AddCommand(initialize.InitCmd)
}
