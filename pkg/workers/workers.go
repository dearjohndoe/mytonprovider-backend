package workers

import (
	"context"
	"log"
	"time"

	providersmaster "mytonprovider-backend/pkg/workers/providersMaster"
	"mytonprovider-backend/pkg/workers/telemetry"
)

type workerFunc = func(ctx context.Context) (interval time.Duration, err error)

type worker struct {
	telemetry       telemetry.Worker
	providersMaster providersmaster.Worker
	logger          *log.Logger
}

type Workers interface {
	Start(ctx context.Context) (err error)
}

func (w *worker) Start(ctx context.Context) (err error) {
	go w.run(ctx, "UpdateTelemetry", w.telemetry.UpdateTelemetry)

	go w.run(ctx, "CollectNewProviders", w.providersMaster.CollectNewProviders)
	go w.run(ctx, "UpdateKnownProviders", w.providersMaster.UpdateKnownProviders)

	return nil
}

func (w *worker) run(ctx context.Context, name string, f workerFunc) {
	for {
		select {
		case <-ctx.Done():
			// Call one last time before exiting
			_, err := f(ctx)
			if err != nil {
				w.logger.Printf("Error in worker %s on exit: %v", name, err)
			} else {
				w.logger.Printf("Worker %s completed successfully on exit", name)
			}
			return
		default:
			interval, err := f(ctx)
			if err != nil {
				w.logger.Printf("Error in worker %s: %v", name, err)
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
	logger *log.Logger,
) Workers {
	return &worker{
		telemetry:       telemetry,
		providersMaster: providersMaster,
		logger:          logger,
	}
}
