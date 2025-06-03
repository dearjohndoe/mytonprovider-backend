package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
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
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() (err error) {
	// Tools
	config := loadConfig()
	if config == nil {
		fmt.Println("Failed to load configuration")
		return
	}

	logger := log.Default()
	telemetryCache := simpleCache.NewSimpleCache(2 * time.Minute)

	// TON
	ton, err := tonclient.NewClient(context.Background(), config.TON.ConfigURL)
	if err != nil {
		logger.Printf("Failed to create TON client: %v", err)
		return
	}

	providerClient, err := newProviderClient(context.Background(), config.TON.ConfigURL, config.System.ADNLPort, config.System.Key)
	if err != nil {
		logger.Printf("Failed to create provider client: %v", err)
		return
	}

	// Postgres
	connPool, err := connectPostgres(context.Background(), config, logger)
	if err != nil {
		logger.Printf("Failed to connect to Postgres: %v", err)
		return
	}

	// Database
	providersRepo := providersRepository.NewRepository(connPool)

	// Workers
	telemetryWorker := telemetry.NewWorker(providersRepo, telemetryCache)
	providersMasterWorker := providersmaster.NewWorker(
		providersRepo,
		ton,
		providerClient,
		config.TON.MasterAddress,
		config.TON.BatchSize,
	)

	cancelCtx, cancel := context.WithCancel(context.Background())
	workers := workers.NewWorkers(telemetryWorker, providersMasterWorker, logger)
	go func() {
		if wErr := workers.Start(cancelCtx); wErr != nil {
			logger.Printf("Failed to start workers: %v", wErr)
			err = wErr
			return
		}
	}()

	// Services
	filesService := providers.NewService(providersRepo, logger)
	filesService = providers.NewCacheMiddleware(filesService, telemetryCache)

	// HTTP Server
	accessTokens := strings.Split(config.System.AccessTokens, ",")
	app := fiber.New()
	server := httpServer.New(app, filesService, accessTokens, logger)

	server.RegisterRoutes()

	go func() {
		if err := app.Listen(":" + config.System.Port); err != nil {
			logger.Printf("Error starting server: %v", err)
		}
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	<-signalChan
	cancel()

	err = app.ShutdownWithTimeout(time.Second * 5)
	if err != nil {
		logger.Printf("Server shut down error %v", err)
		return err
	}

	return err
}
