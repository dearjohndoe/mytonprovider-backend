package httpServer

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"

	v1 "mytonprovider-backend/pkg/models/api/v1"
)

func (h *handler) searchProviders(c *fiber.Ctx) (err error) {
	return c.JSON(okHandler(c))
}

func (h *handler) updateTelemetry(c *fiber.Ctx) (err error) {
	var req v1.TelemetryRequest

	err = json.Unmarshal(c.Body(), &req)
	if err != nil {
		err = fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		return errorHandler(c, err)
	}

	err = h.providers.UpdateTelemetry(c.Context(), &req)
	if err != nil {
		return errorHandler(c, err)
	}

	return c.JSON(okHandler(c))
}

func (h *handler) getLatestTelemetry(c *fiber.Ctx) (err error) {
	// TODO: get TelemetryResponse from service's cache

	return c.JSON(okHandler(c))
}

func (h *handler) health(c *fiber.Ctx) error {
	return c.JSON(okHandler(c))
}
