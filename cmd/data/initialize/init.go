package initialize

import (
	"bytes"
	"context"
	"crypto/tls"
	_ "embed"
	"fmt"
	"net/http"
	"os"
	"time"

	opensearch "github.com/opensearch-project/opensearch-go/v2"
	opensearchapi "github.com/opensearch-project/opensearch-go/v2/opensearchapi"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/gencon_buddy_api/internal/event"
)

var (
	InitCmd = &cobra.Command{
		Use:   "init",
		Short: "Initialize the OpenSearch mapping template and indices.",
		Long:  "Pushes the template mapping using the latest schema. And checks if the relevant indices exist. If not, it creates those.",
		RunE:  run,
	}

	//go:embed schema/event_index_template.json
	eventIndexFile []byte
)

func run(cmd *cobra.Command, _ []string) error {
	logVerbosity := zerolog.InfoLevel
	v, err := cmd.Flags().GetString("verbosity")
	if err != nil {
		return err
	}
	if v != "" {
		logVerbosity, err = zerolog.ParseLevel(v)
		if err != nil {
			return err
		}
	}

	logger := zerolog.New(
		zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339},
	).Level(logVerbosity).With().Timestamp().Caller().Logger()

	osClient, err := opensearch.NewClient(opensearch.Config{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Addresses: []string{"https://localhost:9200"},
		Username:  "admin",
		Password:  "admin",
	})

	if err != nil {
		return err
	}

	x := opensearchapi.IndicesPutIndexTemplateRequest{
		Body: bytes.NewReader(eventIndexFile),
		Name: "event_template",
	}

	response, err := x.Do(context.Background(), osClient)
	if err != nil {
		return err
	}

	fmt.Println(response)
	return event.LoadEventCSV(cmd.Context(), "/mnt/c/workspace/gencon_buddy_api/notes/Gen Con Event Spreadsheets/Gen Con 2021.csv", &logger)
}
