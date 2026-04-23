package main

import (
	"context"
	"fmt"
	"log/slog"
	"mytonprovider-backend/pkg/config"
	"mytonprovider-backend/pkg/metrics"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"

	simpleCache "mytonprovider-backend/pkg/cache"
	"mytonprovider-backend/pkg/clients/ifconfig"
	tonclient "mytonprovider-backend/pkg/clients/ton"
	"mytonprovider-backend/pkg/httpServer"
	providersRepository "mytonprovider-backend/pkg/repositories/providers"
	systemRepository "mytonprovider-backend/pkg/repositories/system"
	"mytonprovider-backend/pkg/services/providers"
	"mytonprovider-backend/pkg/workers"
	"mytonprovider-backend/pkg/workers/cleaner"
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
	cfg := config.MustLoadConfig()
	if cfg == nil {
		fmt.Println("failed to load configuration")
		return
	}

	logLevel := slog.LevelInfo
	if level, ok := config.LogLevels[cfg.System.LogLevel]; ok {
		logLevel = level
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	telemetryCache := simpleCache.NewSimpleCache(2 * time.Minute)
	benchmarksCache := simpleCache.NewSimpleCache(2 * time.Minute)

	// BusinessMetrics
	metrics := metrics.NewBusinessMetrics(metrics.BusinessMetricsConfig{
		Namespace:          cfg.Metrics.Namespace,
		ServerSubSystem:    cfg.Metrics.ServerSubsystem,
		WorkerSubSystem:    cfg.Metrics.WorkersSubsystem,
		DBSubSystem:        cfg.Metrics.DbSubsystem,
		ProvidersSubSystem: cfg.Metrics.ProvidersSubsystem,
	})

	// Clients
	ton, err := tonclient.NewClient(context.Background(), cfg.TON.ConfigURL, logger)
	if err != nil {
		logger.Error("failed to create TON client", slog.String("error", err.Error()))
		return
	}

	ipinfo := ifconfig.NewClient(logger)

	dhtClient, providerClient, err := newProviderClient(context.Background(), cfg.TON.ConfigURL, cfg.System.ADNLPort, cfg.System.Key)
	if err != nil {
		logger.Error("failed to create provider client", slog.String("error", err.Error()))
		return
	}

	// Postgres
	connPool, err := connectPostgres(context.Background(), cfg, logger)
	if err != nil {
		logger.Error("failed to connect to Postgres", slog.String("error", err.Error()))
		return
	}

	// Database
	providersRepo := providersRepository.NewRepository(connPool)
	providersRepo = providersRepository.NewMetrics(metrics.DBRequests, metrics.DBRequestsDuration, providersRepo)

	systemRepo := systemRepository.NewRepository(connPool)
	systemRepo = systemRepository.NewMetrics(metrics.DBRequests, metrics.DBRequestsDuration, systemRepo)

	// Workers
	telemetryWorker := telemetry.NewWorker(providersRepo, telemetryCache, benchmarksCache, metrics.ProvidersNetLoad, logger)
	telemetryWorker = telemetry.NewMetrics(metrics.WorkersRequests, metrics.WorkersRequestsDuration, telemetryWorker)

	providersMasterWorker := providersmaster.NewWorker(
		providersRepo,
		systemRepo,
		ton,
		providerClient,
		dhtClient,
		ipinfo,
		cfg.TON.MasterAddress,
		cfg.TON.BatchSize,
		logger,
	)
	providersMasterWorker = providersmaster.NewMetrics(metrics.WorkersRequests, metrics.WorkersRequestsDuration, providersMasterWorker)

	cleanerWorker := cleaner.NewWorker(providersRepo, cfg.System.StoreHistoryDays, logger)
	cleanerWorker = cleaner.NewMetrics(metrics.WorkersRequests, metrics.WorkersRequestsDuration, cleanerWorker)

	cancelCtx, cancel := context.WithCancel(context.Background())
	workers := workers.NewWorkers(telemetryWorker, providersMasterWorker, cleanerWorker, logger)
	go func() {
		if wErr := workers.Start(cancelCtx); wErr != nil {
			logger.Error("failed to start workers", slog.String("error", wErr.Error()))
			err = wErr
			return
		}
	}()

	// Services
	providersService := providers.NewService(providersRepo, logger)
	providersService = providers.NewCacheMiddleware(providersService, telemetryCache, benchmarksCache)

	// HTTP Server
	accessTokens := strings.Split(cfg.System.AccessTokens, ",")
	app := fiber.New()
	server := httpServer.New(
		app,
		providersService,
		accessTokens,
		cfg.Metrics.Namespace,
		cfg.Metrics.ServerSubsystem,
		logger,
	)

	server.RegisterRoutes()

	go func() {
		if err := app.Listen(":" + cfg.System.Port); err != nil {
			logger.Error("error starting server", slog.String("err", err.Error()))
		}
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	<-signalChan
	cancel()

	err = app.ShutdownWithTimeout(time.Second * 5)
	if err != nil {
		logger.Error("server shut down error", slog.String("err", err.Error()))
		return err
	}

	return err
}
