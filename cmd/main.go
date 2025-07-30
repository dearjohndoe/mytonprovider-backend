package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"

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
	config := loadConfig()
	if config == nil {
		fmt.Println("failed to load configuration")
		return
	}

	logLevel := slog.LevelInfo
	if level, ok := logLevels[config.System.LogLevel]; ok {
		logLevel = level
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	telemetryCache := simpleCache.NewSimpleCache(2 * time.Minute)
	benchmarksCache := simpleCache.NewSimpleCache(10 * time.Minute)

	// Metrics
	dbRequestsCount := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Metrics.Namespace,
			Subsystem: config.Metrics.DbSubsystem,
			Name:      "db_requests_count",
			Help:      "Db requests count",
		},
		[]string{"method", "error"},
	)

	dbRequestsDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.Metrics.Namespace,
			Subsystem: config.Metrics.DbSubsystem,
			Name:      "db_requests_duration",
			Help:      "Db requests duration",
		},
		[]string{"method", "error"},
	)

	workersRunCount := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Metrics.Namespace,
			Subsystem: config.Metrics.DbSubsystem,
			Name:      "workers_requests_count",
			Help:      "Workers requests count",
		},
		[]string{"method", "error"},
	)

	workersRunDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.Metrics.Namespace,
			Subsystem: config.Metrics.DbSubsystem,
			Name:      "workers_requests_duration",
			Help:      "Workers requests duration",
		},
		[]string{"method", "error"},
	)

	prometheus.MustRegister(
		dbRequestsCount,
		dbRequestsDuration,
		workersRunCount,
		workersRunDuration,
	)

	// Clients
	ton, err := tonclient.NewClient(context.Background(), config.TON.ConfigURL, logger)
	if err != nil {
		logger.Error("failed to create TON client", slog.String("error", err.Error()))
		return
	}

	ipinfo := ifconfig.NewClient(logger)
	if err != nil {
		logger.Error("failed to create IP info client", slog.String("error", err.Error()))
		return
	}

	dhtClient, providerClient, err := newProviderClient(context.Background(), config.TON.ConfigURL, config.System.ADNLPort, config.System.Key)
	if err != nil {
		logger.Error("failed to create provider client", slog.String("error", err.Error()))
		return
	}

	// Postgres
	connPool, err := connectPostgres(context.Background(), config, logger)
	if err != nil {
		logger.Error("failed to connect to Postgres", slog.String("error", err.Error()))
		return
	}

	// Database
	providersRepo := providersRepository.NewRepository(connPool)
	providersRepo = providersRepository.NewMetrics(dbRequestsCount, dbRequestsDuration, providersRepo)

	systemRepo := systemRepository.NewRepository(connPool)
	systemRepo = systemRepository.NewMetrics(dbRequestsCount, dbRequestsDuration, systemRepo)

	// Workers
	telemetryWorker := telemetry.NewWorker(providersRepo, telemetryCache, benchmarksCache, logger)
	telemetryWorker = telemetry.NewMetrics(workersRunCount, workersRunDuration, telemetryWorker)

	providersMasterWorker := providersmaster.NewWorker(
		providersRepo,
		systemRepo,
		ton,
		providerClient,
		dhtClient,
		ipinfo,
		config.TON.MasterAddress,
		config.TON.BatchSize,
		logger,
	)
	providersMasterWorker = providersmaster.NewMetrics(workersRunCount, workersRunDuration, providersMasterWorker)

	cleanerWorker := cleaner.NewWorker(providersRepo, config.System.StoreHistoryDays, logger)
	cleanerWorker = cleaner.NewMetrics(workersRunCount, workersRunDuration, cleanerWorker)

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
	accessTokens := strings.Split(config.System.AccessTokens, ",")
	app := fiber.New()
	server := httpServer.New(
		app,
		providersService,
		accessTokens,
		config.Metrics.Namespace,
		config.Metrics.ServerSubsystem,
		logger,
	)

	server.RegisterRoutes()

	go func() {
		if err := app.Listen(":" + config.System.Port); err != nil {
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
