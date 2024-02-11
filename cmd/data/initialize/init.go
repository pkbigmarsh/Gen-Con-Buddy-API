package initialize

import (
	"bytes"
	"context"
	"crypto/tls"
	_ "embed"
	"fmt"
	"net/http"

	opensearch "github.com/opensearch-project/opensearch-go/v2"
	opensearchapi "github.com/opensearch-project/opensearch-go/v2/opensearchapi"
	"github.com/spf13/cobra"
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

func run(_ *cobra.Command, _ []string) error {
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

	return nil
}
