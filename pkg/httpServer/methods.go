package httpServer

import (
	"encoding/json"
	"log/slog"

	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	v1 "mytonprovider-backend/pkg/models/api/v1"
)

func (h *handler) limitReached(c *fiber.Ctx) error {
	log := h.logger.With(
		slog.String("method", "limitReached"),
		slog.String("method", c.Method()),
		slog.String("url", c.OriginalURL()),
		slog.Any("headers", c.GetReqHeaders()),
	)

	log.Warn("rate limit reached for request")
	return fiber.NewError(fiber.StatusTooManyRequests, "too many requests, please try again later")
}

func (h *handler) searchProviders(c *fiber.Ctx) (err error) {
	body := c.Body()
	log := h.logger.With(
		slog.String("method", "searchProviders"),
		slog.String("method", c.Method()),
		slog.String("url", c.OriginalURL()),
		slog.Any("headers", c.GetReqHeaders()),
		slog.Int("body_length", len(body)),
		slog.String("body", string(body)),
	)

	var req v1.SearchProvidersRequest
	err = json.Unmarshal(c.Body(), &req)
	if err != nil {
		log.Error("failed to parse search providers body", slog.String("error", err.Error()))
		err = fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		return errorHandler(c, err)
	}

	providers, err := h.providers.SearchProviders(c.Context(), req)
	if err != nil {
		return errorHandler(c, err)
	}

	return c.JSON(v1.ProvidersResponse{
		Providers: providers,
	})
}

func (h *handler) filtersRange(c *fiber.Ctx) (err error) {
	filters, err := h.providers.GetFiltersRange(c.Context())
	if err != nil {
		return errorHandler(c, err)
	}

	return c.JSON(fiber.Map{
		"filters_range": filters,
	})
}

func (h *handler) updateTelemetry(c *fiber.Ctx) (err error) {
	body := c.Body()
	log := h.logger.With(
		slog.String("method", "updateTelemetry"),
		slog.String("method", c.Method()),
		slog.String("url", c.OriginalURL()),
		slog.Any("headers", c.GetReqHeaders()),
		slog.Int("body_length", len(body)),
		slog.String("body", string(body)),
	)

	if len(body) == 0 || body[0] != '{' {
		err = fiber.NewError(fiber.StatusBadRequest, "invalid gzip body")
		return errorHandler(c, err)
	}

	var req v1.TelemetryRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		log.Error("failed to parse telemetry body", slog.String("error", err.Error()))
		err = fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		return errorHandler(c, err)
	}

	err = h.providers.UpdateTelemetry(c.Context(), &req)
	if err != nil {
		return errorHandler(c, err)
	}

	return c.JSON(okHandler(c))
}

func (h *handler) updateBenchmarks(c *fiber.Ctx) (err error) {
	body := c.Body()
	log := h.logger.With(
		slog.String("method", "updateBenchmarks"),
		slog.String("method", c.Method()),
		slog.String("url", c.OriginalURL()),
		slog.Any("headers", c.GetReqHeaders()),
		slog.Int("body_length", len(body)),
		slog.String("body", string(body)),
	)

	if len(body) == 0 || body[0] != '{' {
		err = fiber.NewError(fiber.StatusBadRequest, "invalid gzip body")
		return errorHandler(c, err)
	}

	var req v1.BenchmarksRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		log.Error("failed to parse benchmarks body", slog.String("error", err.Error()))
		err = fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		return errorHandler(c, err)
	}

	err = h.providers.UpdateBenchmarks(c.Context(), &req)
	if err != nil {
		return errorHandler(c, err)
	}

	return c.JSON(okHandler(c))
}

func (h *handler) getLatestTelemetry(c *fiber.Ctx) (err error) {
	providers, err := h.providers.GetLatestTelemetry(c.Context())
	if err != nil {
		return errorHandler(c, err)
	}

	return c.JSON(fiber.Map{
		"providers": providers,
	})
}

func (h *handler) health(c *fiber.Ctx) error {
	return c.JSON(okHandler(c))
}

func (h *handler) metrics(c *fiber.Ctx) error {
	m := promhttp.Handler()

	return adaptor.HTTPHandler(m)(c)
}
