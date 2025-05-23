package httpServer

import (
	"log"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

type providers interface {
	// AddFile(context.Context) (interface{}, error)
	// AddFolder(context.Context) (interface{}, error)
}

type rateLimiter struct {
	tokens      int
	lastRequest time.Time
}

type errorResponse struct {
	Error string `json:"error"`
}

type handler struct {
	server    *fiber.App
	logger    *log.Logger
	providers providers
	mu        sync.Mutex
	clients   map[string]map[string]rateLimiter // route -> clientIP -> rateLimiter
}

func New(server *fiber.App, providers providers, logger *log.Logger) *handler {
	clients := make(map[string]map[string]rateLimiter)

	return &handler{
		server:    server,
		providers: providers,
		clients:   clients,
		logger:    logger,
	}
}
