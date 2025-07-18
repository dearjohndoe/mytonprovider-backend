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

	m := newMetrics(h.namespace, h.subsystem)

	h.server.Use(m.metricsMiddleware)

	h.server.Use(limiter.New(limiter.Config{
		Max:               MaxRequests,
		Expiration:        RateLimitWindow,
		LimitReached:      h.limitReached,
		LimiterMiddleware: limiter.SlidingWindow{},
	}))

	h.server.Get("/health", h.health)
	h.server.Get("/metrics", h.authorizationMiddleware, h.metrics)

	apiv1 := h.server.Group("/api/v1", h.loggerMiddleware)
	{
		providers := apiv1.Group("/providers")
		providers.Post("/search", h.searchProviders)
		providers.Get("/filters", h.filtersRange)
		providers.Post("", h.updateTelemetry)
		providers.Get("", h.authorizationMiddleware, h.getLatestTelemetry)

		apiv1.Post("/benchmarks", h.updateBenchmarks)
	}
}
