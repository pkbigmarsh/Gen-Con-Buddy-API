package initialize

import (
	_ "embed"
	"errors"
	"fmt"

	"github.com/gencon_buddy_api/cmd/app"
	"github.com/gencon_buddy_api/internal/event"
	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
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

func run(cmd *cobra.Command, _ []string) error {
	filepath, err := cmd.Flags().GetString("filepath")
	if err != nil {
		return fmt.Errorf("failed to read csv filepath: %w", err)
	}

	gcb := app.GetAppFromContext(cmd.Context())
	if gcb == nil {
		return fmt.Errorf("failed to load gcp app context")
	}

	// gcb.Logger.Info().Msg("Updating event index template")
	// x := opensearchapi.IndicesPutIndexTemplateRequest{
	// 	Body: bytes.NewReader(eventIndexFile),
	// 	Name: "event_template",
	// }

	// response, err := x.Do(context.Background(), gcb.OSClient)
	// if err != nil {
	// 	return err
	// }

	// gcb.Logger.Debug().Msgf("Index Template response: %s", response.String())

	clean, err := cmd.Flags().GetBool("clean")
	if err != nil {
		return err
	}

	if clean {
		eventIndex, err := cmd.Flags().GetString("event_index")
		if err != nil {
			return fmt.Errorf("failed to read persistent flag event_index: %s", err)
		}
		eventIndexPattern := eventIndex

		gcb.Logger.Info().Msgf("Cleaning event indicies: %s", eventIndexPattern)
		deleteIndexRequest := opensearchapi.IndicesDeleteRequest{
			Index: []string{eventIndexPattern},
		}

		resp, err := deleteIndexRequest.Do(cmd.Context(), gcb.OSClient)
		if err != nil {
			return fmt.Errorf("failed to deleted event index [%s]: %w", eventIndexPattern, err)
		}

		defer func() {
			if err := resp.Body.Close(); err != nil {
				gcb.Logger.Err(err)
			}
		}()

		if resp.IsError() {
			gcb.Logger.Warn().Msgf("There was a problem deleting indicies for pattern [%s]. Got code [%d]", eventIndexPattern, resp.StatusCode)
			gcb.Logger.Error().Msgf("Raw delete response: %s", resp.String())
		} else {
			gcb.Logger.Debug().Msgf("Debuge delete response: %s", resp.String())
		}
	}
	var evts []*event.Event

	evts, err = event.LoadEventCSV(cmd.Context(), filepath, gcb.Logger)
	if err != nil {
		return err
	}

	eventErrs, err := gcb.EventRepo.WriteEvents(cmd.Context(), evts)
	if len(eventErrs) > 0 {
		gcb.Logger.Error().Msgf("Failed to write events %s", errors.Join(eventErrs...))
	}

	return err
}
