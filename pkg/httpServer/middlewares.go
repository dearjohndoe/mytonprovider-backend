package httpServer

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

const (
	rateLimitDuration = 60
)

func (h *handler) loggerMiddleware(c *fiber.Ctx) error {
	h.logger.Printf("Request: %s %s", c.Method(), c.Path())
	return c.Next()
}

func (h *handler) rateLimiterMiddleware(routeName string, rateLimit int) fiber.Handler {
	h.clients[routeName] = make(map[string]rateLimiter)

	return func(c *fiber.Ctx) error {
		clientIP := c.IP()

		h.mu.Lock()
		defer h.mu.Unlock()

		if _, exists := h.clients[routeName][clientIP]; !exists {
			h.clients[routeName][clientIP] = rateLimiter{
				tokens:      rateLimit,
				lastRequest: time.Now(),
			}
		}

		client := h.clients[routeName][clientIP]
		now := time.Now()

		elapsed := now.Sub(client.lastRequest)
		refill := int(elapsed.Seconds()) * rateLimit / rateLimitDuration
		if refill > 0 {
			client.tokens = min(rateLimit, client.tokens+refill)
			client.lastRequest = now
		}

		if client.tokens > 0 {
			client.tokens--
			h.clients[routeName][clientIP] = client
			return c.Next()
		}

		return errorHandler(c, fiber.NewError(fiber.StatusTooManyRequests, "Too Many Requests"))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}
