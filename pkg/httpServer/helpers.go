package httpServer

import "github.com/gofiber/fiber/v2"

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

	errorResponse := errorResponse{
		Error: err.Error(),
	}

	return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
}
