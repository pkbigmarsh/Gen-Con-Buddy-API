package initialize

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strings"

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

	//go:embed schema/change_log_index.json
	changeLogIndexFile []byte
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
			return fmt.Errorf("failed to read persistent flag event_index: %w", err)
		}

		if err := cleanIndex(cmd.Context(), gcb, eventIndex, eventIndexFile); err != nil {
			return fmt.Errorf("failed to clean and create the event index: %w", err)
		}

		changeLogIndex, err := cmd.Flags().GetString("change_log_index")
		if err != nil {
			return fmt.Errorf("failed to read persistent flag change log index: %w", err)
		}

		if err := cleanIndex(cmd.Context(), gcb, changeLogIndex, changeLogIndexFile); err != nil {
			return fmt.Errorf("failed to clean and create the change log index: %w", err)
		}
	}

	var evts []*event.Event

	if strings.HasSuffix(filepath, ".csv") {
		evts, err = event.LoadEventCSV(cmd.Context(), filepath, gcb.Logger)
	} else if strings.HasSuffix(filepath, ".xlsx") {
		evts, err = event.LoadEventXLSX(cmd.Context(), filepath, gcb.Logger)
	} else {
		return fmt.Errorf("unknown file type in filepath: %s", filepath)
	}

	if err != nil {
		return err
	}

	// When writing the events for the first time, take the `ticketsAvailable`
	// as the total ticket pool
	for _, e := range evts {
		e.TotalTickets = e.TicketsAvailable
	}

	eventErrs, err := gcb.EventRepo.CreateEvents(cmd.Context(), evts)
	if len(eventErrs) > 0 {
		gcb.Logger.Error().Msgf("Failed to write events %s", errors.Join(eventErrs...))
	}

	return err
}

func cleanIndex(ctx context.Context, gcb *app.App, index_name string, index_settings []byte) error {
	gcb.Logger.Info().Msgf("Cleaning index: %s", index_name)
	deleteIndexRequest := opensearchapi.IndicesDeleteRequest{
		Index: []string{index_name},
	}

	resp, err := deleteIndexRequest.Do(ctx, gcb.OSClient)
	if err != nil {
		return fmt.Errorf("failed to delete index [%s]: %w", index_name, err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			gcb.Logger.Err(err)
		}
	}()

	// if the index is not found, then it counts as being deleted
	if resp.IsError() && resp.StatusCode != 404 {
		gcb.Logger.Warn().Msgf("There was a problem deleting idex [%s]. Got code [%d]", index_name, resp.StatusCode)
		gcb.Logger.Error().Msgf("Raw delete response: %s", resp.String())
		return fmt.Errorf("failed to delete indices %s", index_name)
	} else {
		gcb.Logger.Debug().Msgf("Debug delete response: %s", resp.String())
	}

	createIndexRequest := opensearchapi.IndicesCreateRequest{
		Index: index_name,
		Body:  bytes.NewReader(index_settings),
	}

	createResp, createErr := createIndexRequest.Do(ctx, gcb.OSClient)
	if createErr != nil {
		return fmt.Errorf("failed to create index %s: %w", index_name, createErr)
	}

	defer func() {
		if err := createResp.Body.Close(); err != nil {
			gcb.Logger.Err(err)
		}
	}()

	if createResp.IsError() {
		gcb.Logger.Warn().Msgf("There was a problem creating index [%s]. Got code [%d]", index_name, createResp.StatusCode)
		gcb.Logger.Error().Msgf("Raw create response: %s", createResp.String())
		return fmt.Errorf("failed to create index %s", index_name)
	} else {
		gcb.Logger.Debug().Msgf("Debug create response: %s", createResp.String())
	}

	return nil
}
