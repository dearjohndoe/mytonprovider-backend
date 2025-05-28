package httpServer

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"

	v1 "mytonprovider-backend/pkg/models/api/v1"
)

func (h *handler) searchProviders(c *fiber.Ctx) (err error) {
	var req v1.SearchProvidersRequest
	err = json.Unmarshal(c.Body(), &req)
	if err != nil {
		err = fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		return errorHandler(c, err)
	}

	providers, err := h.providers.SearchProviders(c.Context(), req)
	if err != nil {
		return errorHandler(c, err)
	}

	return c.JSON(fiber.Map{
		"providers": providers,
	})
}

func (h *handler) updateTelemetry(c *fiber.Ctx) (err error) {
	defer func() {
		if err != nil {
			h.logger.Println("Error updating telemetry:", err)
		}
	}()

	body := c.Body()
	h.logger.Printf("Request:\nMethod: %s, URL: %s\nHeaders: %v\nBodyLen: %d\nBody:%s",
		c.Method(),
		c.OriginalURL(),
		c.GetReqHeaders(),
		len(body),
		string(body),
	)

	var req v1.TelemetryRequest

	// ignore for now
	// contentEncoding := c.Get("Content-Encoding")
	if len(body) == 0 || body[0] != '{' {
		err = fiber.NewError(fiber.StatusBadRequest, "Invalid gzip body")
		return errorHandler(c, err)
	}

	err = json.Unmarshal(body, &req)
	if err != nil {
		h.logger.Printf("Failed to parse telemetry body. Err: %e", err)
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
