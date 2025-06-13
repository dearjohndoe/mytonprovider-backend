//go:build !debug
// +build !debug

package httpServer

import (
	"time"

	"github.com/gofiber/fiber/v2/middleware/limiter"
)

const (
	MaxRequests     = 100
	RateLimitWindow = 60 * time.Second
)

func (h *handler) RegisterRoutes() {
	h.logger.Info("Registering routes")

	h.server.Use(limiter.New(limiter.Config{
		Max:               MaxRequests,
		Expiration:        RateLimitWindow,
		LimitReached:      h.limitReached,
		LimiterMiddleware: limiter.SlidingWindow{},
	}))

	apiv1 := h.server.Group("/api/v1", h.loggerMiddleware)
	{
		apiv1.Post("/providers/search", h.searchProviders)

		apiv1.Post("/providers", h.updateTelemetry)

		apiv1.Post("/benchmarks", h.updateBenchmarks)

		apiv1.Get("/providers", h.authorizationMiddleware, h.getLatestTelemetry)
	}

	h.server.Get("/health", h.health)
}
