package httpServer

import (
	"crypto/md5"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func (h *handler) authorizationMiddleware(c *fiber.Ctx) error {
	accessToken := c.Get("Authorization")
	if accessToken == "" {
		return errorHandler(c, fiber.NewError(fiber.StatusUnauthorized, "unauthorized"))
	}

	if strings.HasPrefix(strings.ToLower(accessToken), "bearer ") {
		accessToken = accessToken[7:]
	}

	hash := md5.Sum([]byte(accessToken))
	tokenHash := fmt.Sprintf("%x", hash[:])

	if _, exists := h.accessTokens[tokenHash]; !exists {
		return errorHandler(c, fiber.NewError(fiber.StatusForbidden, "forbidden"))
	}

	return c.Next()
}

func (h *handler) loggerMiddleware(c *fiber.Ctx) error {
	headers := c.GetReqHeaders()
	if _, ok := headers["Authorization"]; ok {
		headers["Authorization"] = []string{"REDACTED"}
	}

	h.logger.Debug(
		"request received",
		"method", c.Method(),
		"url", c.OriginalURL(),
		"headers", headers,
		"body_length", len(c.Body()),
	)

	return c.Next()
}
