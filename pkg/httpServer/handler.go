package httpServer

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"

	v1 "mytonprovider-backend/pkg/models/api/v1"
)

type providers interface {
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
	server    *fiber.App
	logger    *log.Logger
	providers providers
	mu        sync.Mutex
	clients   map[string]map[string]rateLimiter // route -> clientIP -> rateLimiter
}

func New(server *fiber.App, providers providers, logger *log.Logger) *handler {
	clients := make(map[string]map[string]rateLimiter)

	return &handler{
		server:    server,
		providers: providers,
		clients:   clients,
		logger:    logger,
	}
}
