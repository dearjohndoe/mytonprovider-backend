package httpServer

import (
	"context"
	"log/slog"

	"github.com/gofiber/fiber/v2"

	v1 "mytonprovider-backend/pkg/models/api/v1"
)

type providers interface {
	SearchProviders(ctx context.Context, req v1.SearchProvidersRequest) ([]v1.Provider, error)
	GetLatestTelemetry(ctx context.Context) (providers []v1.TelemetryRequest, err error)
	UpdateTelemetry(ctx context.Context, telemetry *v1.TelemetryRequest) (err error)
	UpdateBenchmarks(ctx context.Context, benchmark *v1.BenchmarksRequest) (err error)
}

type errorResponse struct {
	Error string `json:"error"`
}

type handler struct {
	server       *fiber.App
	logger       *slog.Logger
	providers    providers
	accessTokens map[string]struct{}
}

func New(
	server *fiber.App,
	providers providers,
	accessTokens []string,
	logger *slog.Logger,
) *handler {
	accessTokensMap := make(map[string]struct{})
	for _, token := range accessTokens {
		accessTokensMap[token] = struct{}{}
	}

	h := &handler{
		server:       server,
		providers:    providers,
		accessTokens: accessTokensMap,
		logger:       logger,
	}

	return h
}
