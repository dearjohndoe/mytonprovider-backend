package cleaner

import (
	"context"
	"log/slog"
	"time"
)

type repository interface {
	CleanOldProvidersHistory(ctx context.Context, days int) (removed int, err error)
	CleanOldStatusesHistory(ctx context.Context, days int) (removed int, err error)
	CleanOldBenchmarksHistory(ctx context.Context, days int) (removed int, err error)
	CleanOldTelemetryHistory(ctx context.Context, days int) (removed int, err error)
}

type cleanerWorker struct {
	repo   repository
	days   int
	logger *slog.Logger
}

type Worker interface {
	CleanupOldData(ctx context.Context) (interval time.Duration, err error)
}

func (w *cleanerWorker) CleanupOldData(ctx context.Context) (interval time.Duration, err error) {
	const (
		failureInterval = 5 * time.Second
		successInterval = 1 * time.Hour
	)

	log := w.logger.With("worker", "CleanupOldData")
	log.Debug("cleaning up old data")

	interval = successInterval

	if removed, err := w.repo.CleanOldProvidersHistory(ctx, w.days); err != nil {
		log.Error("failed to clean old providers history", slog.Int("days", w.days), slog.String("err", err.Error()))
		interval = failureInterval
	} else if removed > 0 {
		log.Info("cleaned old providers history", slog.Int("removed", removed))
	}

	if removed, err := w.repo.CleanOldStatusesHistory(ctx, w.days); err != nil {
		log.Error("failed to clean old statuses history", slog.Int("days", w.days), slog.String("err", err.Error()))
		interval = failureInterval
	} else if removed > 0 {
		log.Info("cleaned old statuses history", slog.Int("removed", removed))
	}

	if removed, err := w.repo.CleanOldBenchmarksHistory(ctx, w.days); err != nil {
		log.Error("failed to clean old benchmarks history", slog.Int("days", w.days), slog.String("err", err.Error()))
		interval = failureInterval
	} else if removed > 0 {
		log.Info("cleaned old benchmarks history", slog.Int("removed", removed))
	}

	if removed, err := w.repo.CleanOldTelemetryHistory(ctx, w.days); err != nil {
		log.Error("failed to clean old telemetry history", slog.Int("days", w.days), slog.String("err", err.Error()))
		interval = failureInterval
	} else if removed > 0 {
		log.Info("cleaned old telemetry history", slog.Int("removed", removed))
	}

	return
}

func NewWorker(repo repository, days int, logger *slog.Logger) Worker {
	return &cleanerWorker{
		repo:   repo,
		days:   days,
		logger: logger,
	}
}
