package initialize

import (
	"bytes"
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
			return fmt.Errorf("failed to delete event index [%s]: %w", eventIndexPattern, err)
		}

		defer func() {
			if err := resp.Body.Close(); err != nil {
				gcb.Logger.Err(err)
			}
		}()

		// if the index is not found, then it counts as being deleted
		if resp.IsError() && resp.StatusCode != 404 {
			gcb.Logger.Warn().Msgf("There was a problem deleting indicies for pattern [%s]. Got code [%d]", eventIndexPattern, resp.StatusCode)
			gcb.Logger.Error().Msgf("Raw delete response: %s", resp.String())
			return fmt.Errorf("failed to delete indices %s", eventIndexPattern)
		} else {
			gcb.Logger.Debug().Msgf("Debuge delete response: %s", resp.String())
		}

		createIndexRequest := opensearchapi.IndicesCreateRequest{
			Index: eventIndex,
			Body:  bytes.NewReader(eventIndexFile),
		}

		createResp, createErr := createIndexRequest.Do(cmd.Context(), gcb.OSClient)
		if createErr != nil {
			return fmt.Errorf("failed to create event index %s: %w", eventIndex, createErr)
		}

		defer func() {
			if err := createResp.Body.Close(); err != nil {
				gcb.Logger.Err(err)
			}
		}()

		if createResp.IsError() {
			gcb.Logger.Warn().Msgf("There was a problem creating index [%s]. Got code [%d]", eventIndex, createResp.StatusCode)
			gcb.Logger.Error().Msgf("Raw create response: %s", createResp.String())
			return fmt.Errorf("failed to create index %s", eventIndex)
		} else {
			gcb.Logger.Debug().Msgf("Debuge create response: %s", createResp.String())
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
