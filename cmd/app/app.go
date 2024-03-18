package app

import (
	"context"
	"crypto/tls"
	"net/http"

	"github.com/gencon_buddy_api/internal/event"
	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/rs/zerolog"
)

type appContextKey uint

const (
	appKey appContextKey = iota
)

// App holds relevant information for the gcb cli app
type App struct {
	Logger    zerolog.Logger
	OSClient  *opensearch.Client
	EventRepo *event.EventRepo
}

// AppConfig contains all configuration needed to initialize the GCB App
type AppConfig struct {
	OSAddress  string
	OSUsername string
	OSPassword string
	EventIndex string
	BatchSize  int
}

// NewApp initializes the shared GCP App
func NewApp(logger zerolog.Logger, config AppConfig) (*App, error) {
	logger.Debug().Msgf("Initializing GCB App with config: %v", config)
	client, err := opensearch.NewClient(opensearch.Config{
		Addresses: []string{config.OSAddress},
		Username:  config.OSUsername,
		Password:  config.OSPassword,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	})

	if err != nil {
		return nil, err
	}

	return &App{
		Logger:    logger,
		OSClient:  client,
		EventRepo: event.NewEventRepo(&logger, client, config.BatchSize, config.EventIndex),
	}, nil
}

// GetAppFromContext fetches the App struct from the context if it exists
func GetAppFromContext(ctx context.Context) *App {
	gcb := ctx.Value(appKey)
	if gcb == nil {
		return nil
	}

	return gcb.(*App)
}

// ContextWithApp creates a new [context.Context] from the parentContext with the App added to it
func ContextWithApp(parentContext context.Context, app *App) context.Context {
	return context.WithValue(parentContext, appKey, app)
}
