package httpServer

import (
	"github.com/gofiber/fiber/v2"

	"mytonprovider-backend/pkg/models"
)

func okHandler(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "ok",
	})
}

func errorHandler(c *fiber.Ctx, err error) error {
	if e, ok := err.(*fiber.Error); ok {
		return c.Status(e.Code).JSON(fiber.Map{
			"error": e.Message,
		})
	}

	if appErr, ok := err.(*models.AppError); ok {
		msg := appErr.Message
		if appErr.Code > 500 {
			msg = "internal server error"
		}

		return c.Status(appErr.Code).JSON(fiber.Map{
			"error": msg,
		})
	}

	errorResponse := errorResponse{
		Error: err.Error(),
	}

	return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
}
