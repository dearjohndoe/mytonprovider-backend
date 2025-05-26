package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"

	simpleCache "mytonprovider-backend/pkg/cache"
	"mytonprovider-backend/pkg/httpServer"
	providersRepository "mytonprovider-backend/pkg/repositories/providers"
	"mytonprovider-backend/pkg/services/providers"
	"mytonprovider-backend/pkg/tonclient"
	"mytonprovider-backend/pkg/workers"
	providersmaster "mytonprovider-backend/pkg/workers/providersMaster"
	"mytonprovider-backend/pkg/workers/telemetry"
)

func main() {
	// Tools
	config := loadConfig()
	if config == nil {
		fmt.Println("Failed to load configuration")
		return
	}

	logger := log.Default()
	telemetryCache := simpleCache.NewSimpleCache(2 * time.Minute)

	ctx := context.Background()

	// TON
	ton, err := tonclient.NewClient(ctx, config.TON.ConfigURL)
	if err != nil {
		logger.Printf("Failed to create TON client: %v", err)
		return
	}

	// Postgres
	connPool, err := connectPostgres(config, logger)
	if err != nil {
		logger.Printf("Failed to connect to Postgres: %v", err)
		return
	}
	defer connPool.Close()

	// Database
	providersRepo := providersRepository.NewRepository(connPool)

	// Workers
	telemetryWorker := telemetry.NewWorker(providersRepo, telemetryCache)
	providersMasterWorker := providersmaster.NewWorker(providersRepo, ton, config.TON.MasterAddress, config.TON.BatchSize)
	workers := workers.NewWorkers(telemetryWorker, providersMasterWorker, logger)
	if err := workers.Start(ctx); err != nil {
		logger.Printf("Failed to start workers: %v", err)
		return
	}

	// Services
	filesService := providers.NewService(providersRepo, logger)
	filesService = providers.NewCacheMiddleware(filesService, telemetryCache)

	// HTTP Server
	app := fiber.New()
	server := httpServer.New(app, filesService, logger)

	server.RegisterRoutes()

	// Gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := app.Listen(":" + config.System.Port); err != nil {
			logger.Printf("Error starting server: %v", err)
		}
	}()

	<-quit
	logger.Println("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		logger.Printf("Error during server shutdown: %v", err)
	}

	logger.Println("Server stopped.")
}
