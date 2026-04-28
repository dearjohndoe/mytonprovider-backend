package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"

	simpleCache "mytonprovider-backend/pkg/cache"
	"mytonprovider-backend/pkg/config"
	"mytonprovider-backend/pkg/dispatcher"
	"mytonprovider-backend/pkg/handlers"
	"mytonprovider-backend/pkg/httpServer"
	"mytonprovider-backend/pkg/jobs"
	"mytonprovider-backend/pkg/metrics"
	"mytonprovider-backend/pkg/redisstream"
	providersRepository "mytonprovider-backend/pkg/repositories/providers"
	systemRepository "mytonprovider-backend/pkg/repositories/system"
	"mytonprovider-backend/pkg/services/providers"
	"mytonprovider-backend/pkg/workers"
	"mytonprovider-backend/pkg/workers/cleaner"
	"mytonprovider-backend/pkg/workers/telemetry"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() (err error) {
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

	businessMetrics := metrics.NewBusinessMetrics(metrics.BusinessMetricsConfig{
		Namespace:          cfg.Metrics.Namespace,
		ServerSubSystem:    cfg.Metrics.ServerSubsystem,
		WorkerSubSystem:    cfg.Metrics.WorkersSubsystem,
		DBSubSystem:        cfg.Metrics.DbSubsystem,
		ProvidersSubSystem: cfg.Metrics.ProvidersSubsystem,
	})

	// Postgres
	connPool, err := connectPostgres(context.Background(), cfg, logger)
	if err != nil {
		logger.Error("failed to connect to Postgres", slog.String("error", err.Error()))
		return
	}
	defer connPool.Close()

	// Repositories
	providersRepo := providersRepository.NewRepository(connPool)
	providersRepo = providersRepository.NewMetrics(businessMetrics.DBRequests, businessMetrics.DBRequestsDuration, providersRepo)

	systemRepo := systemRepository.NewRepository(connPool)
	systemRepo = systemRepository.NewMetrics(businessMetrics.DBRequests, businessMetrics.DBRequestsDuration, systemRepo)

	// Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer rdb.Close()

	if pErr := rdb.Ping(context.Background()).Err(); pErr != nil {
		logger.Error("failed to ping Redis", slog.String("error", pErr.Error()))
		err = pErr
		return
	}

	publisher := redisstream.NewPublisher(rdb, cfg.Redis.StreamPrefix, cfg.Redis.TriggerMaxLen)

	// Cycle bindings (cycle_type → schedule + handler)
	cycleSet := handlers.NewSet(logger, providersRepo, systemRepo)

	type binding struct {
		cycleType string
		schedule  config.CycleSchedule
	}

	bindings := []binding{
		{jobs.CycleScanMaster, cfg.Cycles.ScanMaster},
		{jobs.CycleScanWallets, cfg.Cycles.ScanWallets},
		{jobs.CycleResolveEndpoints, cfg.Cycles.ResolveEndpoints},
		{jobs.CycleProbeRates, cfg.Cycles.ProbeRates},
		{jobs.CycleInspectContracts, cfg.Cycles.InspectContracts},
		{jobs.CycleCheckProofs, cfg.Cycles.CheckProofs},
		{jobs.CycleLookupIPInfo, cfg.Cycles.LookupIPInfo},
	}

	// Result-streams: EnsureGroup + spawn consumer per type
	cancelCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var consumersWg sync.WaitGroup

	for _, b := range bindings {
		if !b.schedule.Enabled {
			continue
		}

		resultStream := fmt.Sprintf("%s:result:%s", cfg.Redis.StreamPrefix, b.cycleType)
		if eErr := redisstream.EnsureGroup(cancelCtx, rdb, resultStream, cfg.Redis.Group); eErr != nil {
			logger.Error("ensure result group", slog.String("cycle", b.cycleType), slog.String("error", eErr.Error()))
			err = eErr
			return
		}

		handler := cycleSet.Handler(b.cycleType)
		if handler == nil {
			logger.Error("no handler for cycle", slog.String("cycle", b.cycleType))
			continue
		}

		consumer := redisstream.NewConsumer(rdb, connPool, systemRepo, redisstream.ConsumerConfig{
			Stream:     resultStream,
			Group:      cfg.Redis.Group,
			ConsumerID: "backend",
			CycleType:  b.cycleType,
			Parallel:   cfg.Redis.Parallel,
			BlockMs:    cfg.Redis.BlockMs,
		}, handler, logger)

		consumersWg.Add(1)
		go func(c *redisstream.Consumer, name string) {
			defer consumersWg.Done()
			c.Run(cancelCtx)
		}(consumer, b.cycleType)

		logger.Info("result consumer started", slog.String("cycle", b.cycleType))
	}

	// Dispatcher (cron-like trigger publisher)
	schedules := make([]dispatcher.CycleSchedule, 0, len(bindings))
	for _, b := range bindings {
		schedules = append(schedules, dispatcher.CycleSchedule{
			CycleType:      b.cycleType,
			Enabled:        b.schedule.Enabled,
			Interval:       b.schedule.Interval,
			SingleInflight: b.schedule.SingleInflight,
		})
	}

	disp := dispatcher.New(rdb, publisher, cfg.Redis.Group, schedules, logger)

	consumersWg.Add(1)
	go func() {
		defer consumersWg.Done()
		disp.Run(cancelCtx)
	}()

	// Telemetry/cleaner workers (по-прежнему живут локально)
	telemetryWorker := telemetry.NewWorker(providersRepo, telemetryCache, benchmarksCache, businessMetrics.ProvidersNetLoad, logger)
	telemetryWorker = telemetry.NewMetrics(businessMetrics.WorkersRequests, businessMetrics.WorkersRequestsDuration, telemetryWorker)

	cleanerWorker := cleaner.NewWorker(providersRepo, cfg.System.StoreHistoryDays, logger)
	cleanerWorker = cleaner.NewMetrics(businessMetrics.WorkersRequests, businessMetrics.WorkersRequestsDuration, cleanerWorker)

	localWorkers := workers.NewWorkers(telemetryWorker, cleanerWorker, logger)
	go func() {
		if wErr := localWorkers.Start(cancelCtx); wErr != nil {
			logger.Error("failed to start local workers", slog.String("error", wErr.Error()))
		}
	}()

	// Aggregates worker (uptime/rating через SQL)
	go runAggregates(cancelCtx, providersRepo, logger)

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
		if lErr := app.Listen(":" + cfg.System.Port); lErr != nil {
			logger.Error("error starting server", slog.String("err", lErr.Error()))
		}
	}()

	// Shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	<-signalChan
	logger.Info("shutdown signal received")
	cancel()

	if sErr := app.ShutdownWithTimeout(time.Second * 5); sErr != nil {
		logger.Error("server shut down error", slog.String("err", sErr.Error()))
	}

	// дочитываем in-flight consumers
	consumersWg.Wait()
	logger.Info("backend stopped")
	return nil
}

// runAggregates периодически пересчитывает uptime/rating через SQL.
func runAggregates(
	ctx context.Context,
	repo providersRepository.Repository,
	logger *slog.Logger,
) {
	const interval = 5 * time.Minute

	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		if uErr := repo.UpdateUptime(ctx); uErr != nil {
			logger.Error("update uptime", slog.String("error", uErr.Error()))
		}
		if rErr := repo.UpdateRating(ctx); rErr != nil {
			logger.Error("update rating", slog.String("error", rErr.Error()))
		}

		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}
