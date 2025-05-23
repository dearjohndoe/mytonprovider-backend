package httpServer

import (
	"github.com/gofiber/fiber/v2"
)

type Metadata struct {
	Description string `json:"description"`
}

type FolderMetadata struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (h *handler) getProviders(c *fiber.Ctx) (err error) {
	return c.JSON(okHandler(c))
}

func (h *handler) health(c *fiber.Ctx) error {
	return c.JSON(okHandler(c))
}
