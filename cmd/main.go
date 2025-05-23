package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/patrickmn/go-cache"

	"mytonprovider-backend/pkg/httpServer"
	"mytonprovider-backend/pkg/services/providers"
)

func main() {
	logger := log.Default()

	telemetryCache := cache.New(1*time.Minute, 2*time.Minute)

	filesService := providers.NewService(telemetryCache, logger)

	app := fiber.New()
	server := httpServer.New(app, filesService, logger)

	server.RegisterRoutes()

	// Gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := app.Listen(":9090"); err != nil {
			logger.Printf("Error starting server: %v", err)
		}
	}()

	<-quit
	logger.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		logger.Printf("Error during server shutdown: %v", err)
	}

	logger.Println("Server stopped.")
}
