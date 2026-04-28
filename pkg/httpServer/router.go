//go:build !debug
// +build !debug

package httpServer

import (
	"mytonprovider-backend/pkg/metrics"
	"time"

	"github.com/gofiber/fiber/v2/middleware/limiter"
)

const (
	MaxRequests     = 100
	RateLimitWindow = 60 * time.Second
)

func (h *handler) RegisterRoutes() {
	h.logger.Info("Registering routes")

	m := metrics.NewNetMetrics(metrics.NetMetricsConfig{
		Namespace: h.namespace,
		SubSystem: h.subsystem,
	})

	metricsMiddleware := NewMetricsMiddleware(m)
	h.server.Use(metricsMiddleware)

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
		{
			providers := apiv1.Group("/providers")
			providers.Post("/search", h.searchProviders)
			providers.Get("/filters", h.filtersRange)
			providers.Post("", h.updateTelemetry)
			providers.Get("", h.authorizationMiddleware, h.getLatestTelemetry)
		}

		{
			contracts := apiv1.Group("/contracts")
			contracts.Post("/statuses", h.getStorageContractsStatuses)
		}

		apiv1.Post("/benchmarks", h.updateBenchmarks)
	}
}
