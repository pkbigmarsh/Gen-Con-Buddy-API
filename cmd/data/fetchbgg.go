package data

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/gencon_buddy_api/cmd/app"
	"github.com/gencon_buddy_api/internal/bgg"
)

const (
	flagBGGUsername string = "bgg-username"
	flagBGGPassword string = "bgg-password"
	envBGGUsername  string = "BGG_USERNAME"
	envBGGPassword  string = "BGG_PASSWORD"
)

var fetchBggCmd = &cobra.Command{
	Use:   "fetch-bgg",
	Short: "Download the BGG board game ranks data dump to a CSV file",
	Long:  "Logs into BoardGameGeek, downloads the latest board game ranks data dump, and writes the unzipped CSV to --output.",
	RunE:  fetchBgg,
}

func init() {
	fetchBggCmd.Flags().StringP(outputFlag, "o", "", "path to write the downloaded BGG ranks CSV (required)")
	fetchBggCmd.Flags().String(flagBGGUsername, "", "BGG username (overrides BGG_USERNAME)")
	fetchBggCmd.Flags().String(flagBGGPassword, "", "BGG password (overrides BGG_PASSWORD)")
}

func fetchBgg(cmd *cobra.Command, _ []string) error {
	gcb := app.GetAppFromContext(cmd.Context())
	if gcb == nil {
		return fmt.Errorf("couldn't initialize gcb app context")
	}

	output, err := cmd.Flags().GetString(outputFlag)
	if err != nil {
		return fmt.Errorf("failed to read %s flag: %w", outputFlag, err)
	}

	if output == "" {
		return fmt.Errorf("--%s is required", outputFlag)
	}

	creds, err := resolveBGGCredentials(cmd)
	if err != nil {
		return err
	}

	fetcher, err := bgg.NewFetcher()
	if err != nil {
		return fmt.Errorf("failed to create bgg fetcher: %w", err)
	}

	gcb.Logger.Info().Str("output", output).Msg("Fetching BGG ranks data dump")
	rc, err := fetcher.FetchRanksCSV(cmd.Context(), creds)
	if err != nil {
		return fmt.Errorf("failed to fetch bgg ranks dump: %w", err)
	}

	defer func() {
		if err := rc.Close(); err != nil {
			gcb.Logger.Err(err).Msg("failed to close bgg csv stream")
		}
	}()

	out, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("failed to create output file [%s]: %w", output, err)
	}

	defer func() {
		if err := out.Close(); err != nil {
			gcb.Logger.Err(err).Str("output", output).Msg("failed to close output file")
		}
	}()

	n, err := io.Copy(out, rc)
	if err != nil {
		return fmt.Errorf("failed to write bgg csv to [%s]: %w", output, err)
	}

	gcb.Logger.Info().Int64("bytes", n).Str("output", output).Msg("BGG ranks CSV written")
	return nil
}

// resolveBGGCredentials reads credentials from the --bgg-username/--bgg-password
// flags, falling back to the BGG_USERNAME/BGG_PASSWORD environment variables.
// Returns an error naming the first missing value.
func resolveBGGCredentials(cmd *cobra.Command) (bgg.Credentials, error) {
	username, err := cmd.Flags().GetString(flagBGGUsername)
	if err != nil {
		return bgg.Credentials{}, fmt.Errorf("failed to read %s flag: %w", flagBGGUsername, err)
	}

	if username == "" {
		username = os.Getenv(envBGGUsername)
	}

	password, err := cmd.Flags().GetString(flagBGGPassword)
	if err != nil {
		return bgg.Credentials{}, fmt.Errorf("failed to read %s flag: %w", flagBGGPassword, err)
	}

	if password == "" {
		password = os.Getenv(envBGGPassword)
	}

	if username == "" {
		return bgg.Credentials{}, fmt.Errorf("BGG username not set; provide --%s or %s", flagBGGUsername, envBGGUsername)
	}

	if password == "" {
		return bgg.Credentials{}, fmt.Errorf("BGG password not set; provide --%s or %s", flagBGGPassword, envBGGPassword)
	}

	return bgg.Credentials{Username: username, Password: password}, nil
}
