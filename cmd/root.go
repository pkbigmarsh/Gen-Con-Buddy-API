package cmd

import (
	"fmt"
	"os"

	"github.com/gencon_buddy_api/cmd/api"
	"github.com/gencon_buddy_api/cmd/data"
	"github.com/spf13/cobra"
)

var gcbRootCmd = &cobra.Command{
	Use:   "gcb",
	Short: "GenConBuddy is the cli helper for initiating, setting up, and maintaining the GenConBuddy API Service.",
}

func init() {
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
