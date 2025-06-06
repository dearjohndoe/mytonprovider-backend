package httpServer

import (
	"crypto/md5"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func (h *handler) authorizationMiddleware(c *fiber.Ctx) error {
	accessToken := c.Get("Authorization")
	if accessToken == "" {
		return errorHandler(c, fiber.NewError(fiber.StatusUnauthorized, "unauthorized"))
	}

	hash := md5.Sum([]byte(accessToken))
	tokenHash := fmt.Sprintf("%x", hash[:])

	if _, exists := h.accessTokens[tokenHash]; !exists {
		return errorHandler(c, fiber.NewError(fiber.StatusForbidden, "forbidden"))
	}

	return c.Next()
}

func (h *handler) loggerMiddleware(c *fiber.Ctx) error {
	h.logger.Debug(
		"request received",
		"method", c.Method(),
		"url", c.OriginalURL(),
		"headers", c.GetReqHeaders(),
		"body_length", len(c.Body()),
	)

	return c.Next()
}
