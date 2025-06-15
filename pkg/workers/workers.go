package workers

import (
	"context"
	"log/slog"
	"time"

	"mytonprovider-backend/pkg/workers/cleaner"
	providersmaster "mytonprovider-backend/pkg/workers/providersMaster"
	"mytonprovider-backend/pkg/workers/telemetry"
)

type workerFunc = func(ctx context.Context) (interval time.Duration, err error)

type worker struct {
	telemetry       telemetry.Worker
	providersMaster providersmaster.Worker
	cleaner         cleaner.Worker
	logger          *slog.Logger
}

type Workers interface {
	Start(ctx context.Context) (err error)
}

func (w *worker) Start(ctx context.Context) (err error) {
	go w.run(ctx, "UpdateTelemetry", w.telemetry.UpdateTelemetry)
	go w.run(ctx, "UpdateBenchmarks", w.telemetry.UpdateBenchmarks)

	go w.run(ctx, "CollectNewProviders", w.providersMaster.CollectNewProviders)
	go w.run(ctx, "UpdateKnownProviders", w.providersMaster.UpdateKnownProviders)
	go w.run(ctx, "UpdateUptime", w.providersMaster.UpdateUptime)
	go w.run(ctx, "UpdateRating", w.providersMaster.UpdateRating)

	go w.run(ctx, "CleanupOldData", w.cleaner.CleanupOldData)

	return nil
}

func (w *worker) run(ctx context.Context, name string, f workerFunc) {
	logger := w.logger.With(slog.String("run_worker", name))

	for {
		select {
		case <-ctx.Done():
			return
		default:
			interval, err := f(ctx)
			if err != nil {
				logger.Error(err.Error())
			}
			if interval <= 0 {
				interval = time.Second
			}
			t := time.NewTimer(interval)
			select {
			case <-ctx.Done():
				t.Stop()
				return
			case <-t.C:
			}
		}
	}
}

func NewWorkers(
	telemetry telemetry.Worker,
	providersMaster providersmaster.Worker,
	cleaner cleaner.Worker,
	logger *slog.Logger,
) Workers {
	return &worker{
		telemetry:       telemetry,
		providersMaster: providersMaster,
		cleaner:         cleaner,
		logger:          logger,
	}
}
