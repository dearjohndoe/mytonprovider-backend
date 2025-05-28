package httpServer

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"

	v1 "mytonprovider-backend/pkg/models/api/v1"
	"mytonprovider-backend/pkg/models/db"
)

type providers interface {
	SearchProviders(ctx context.Context, req v1.SearchProvidersRequest) ([]db.Provider, error)
	GetLatestTelemetry(ctx context.Context) (providers []*v1.TelemetryRequest, err error)
	UpdateTelemetry(ctx context.Context, telemetry *v1.TelemetryRequest) (err error)
}

type rateLimiter struct {
	tokens      int
	lastRequest time.Time
}

type errorResponse struct {
	Error string `json:"error"`
}

type handler struct {
	server       *fiber.App
	logger       *log.Logger
	providers    providers
	mu           sync.Mutex
	accessTokens map[string]struct{}
	clients      map[string]map[string]rateLimiter // route -> clientIP -> rateLimiter
}

func New(server *fiber.App, providers providers, accessTokens []string, logger *log.Logger) *handler {
	clients := make(map[string]map[string]rateLimiter)

	accessTokensMap := make(map[string]struct{})
	for _, token := range accessTokens {
		accessTokensMap[token] = struct{}{}
	}

	return &handler{
		server:       server,
		providers:    providers,
		clients:      clients,
		accessTokens: accessTokensMap,
		logger:       logger,
	}
}
