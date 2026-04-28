package redisstream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"mytonprovider-backend/pkg/jobs"
	systemRepo "mytonprovider-backend/pkg/repositories/system"
)

// ResultHandler — обработчик одного результата. Вызывается в рамках транзакции
// уже после успешной dedup-вставки в system.processed_jobs. Должен только
// применить writes к БД и вернуться без ошибки. XACK сделается caller'ом
// после COMMIT.
type ResultHandler func(ctx context.Context, tx pgx.Tx, env jobs.ResultEnvelope) error

// Consumer слушает один result-stream через consumer-group.
// На каждое сообщение:
//  1. BEGIN tx
//  2. system.MarkProcessedTx — если дубль, COMMIT (пустая) и XACK
//  3. handler в этой же tx
//  4. COMMIT → XACK; при ошибке ROLLBACK без XACK (сообщение переедет)
type Consumer struct {
	rdb        *redis.Client
	pool       *pgxpool.Pool
	systemRepo systemRepo.Repository

	stream     string
	group      string
	consumerID string

	cycleType string
	parallel  int
	blockMs   int

	handler ResultHandler
	logger  *slog.Logger
}

type ConsumerConfig struct {
	Stream     string
	Group      string
	ConsumerID string
	CycleType  string
	Parallel   int
	BlockMs    int
}

func NewConsumer(
	rdb *redis.Client,
	pool *pgxpool.Pool,
	systemRepo systemRepo.Repository,
	cfg ConsumerConfig,
	handler ResultHandler,
	logger *slog.Logger,
) *Consumer {
	if cfg.Parallel <= 0 {
		cfg.Parallel = 1
	}
	if cfg.BlockMs <= 0 {
		cfg.BlockMs = 5000
	}
	return &Consumer{
		rdb:        rdb,
		pool:       pool,
		systemRepo: systemRepo,
		stream:     cfg.Stream,
		group:      cfg.Group,
		consumerID: cfg.ConsumerID,
		cycleType:  cfg.CycleType,
		parallel:   cfg.Parallel,
		blockMs:    cfg.BlockMs,
		handler:    handler,
		logger:     logger,
	}
}

func (c *Consumer) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for i := 0; i < c.parallel; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			c.runWorker(ctx, idx)
		}(i)
	}
	wg.Wait()
}

func (c *Consumer) runWorker(ctx context.Context, idx int) {
	log := c.logger.With(
		slog.String("cycle", c.cycleType),
		slog.String("consumer", c.consumerID),
		slog.Int("worker", idx),
	)
	log.Debug("result consumer started")

	for {
		if ctx.Err() != nil {
			return
		}

		streams, err := c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    c.group,
			Consumer: c.consumerID,
			Streams:  []string{c.stream, ">"},
			Count:    1,
			Block:    time.Duration(c.blockMs) * time.Millisecond,
		}).Result()

		if err != nil {
			if errors.Is(err, redis.Nil) || errors.Is(err, context.Canceled) {
				continue
			}
			if ctx.Err() != nil {
				return
			}
			log.Error("xreadgroup", slog.String("error", err.Error()))
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
			continue
		}

		for _, s := range streams {
			for _, msg := range s.Messages {
				c.process(ctx, log, msg)
			}
		}
	}
}

func (c *Consumer) process(ctx context.Context, log *slog.Logger, msg redis.XMessage) {
	env, err := decodeEnvelope(msg)
	if err != nil {
		log.Error("malformed result", slog.String("msg_id", msg.ID), slog.String("error", err.Error()))
		// XACK — сообщение нечитаемо, не повторим
		c.ack(ctx, log, msg.ID)
		return
	}

	log = log.With(
		slog.String("msg_id", msg.ID),
		slog.String("job_id", env.JobID),
		slog.String("agent", env.AgentID),
		slog.String("status", env.Status),
	)

	if env.Status == jobs.StatusError {
		log.Warn("agent reported error", slog.String("error", env.Error))
		// fall through — всё равно дедупим и acknowledge'м
	}

	committed, err := c.applyTx(ctx, env)
	if err != nil {
		log.Error("apply tx failed", slog.String("error", err.Error()))
		// БЕЗ XACK — сообщение переедет, попробуем снова
		return
	}

	if committed {
		log.Info("result applied")
	} else {
		log.Debug("duplicate result, skipped")
	}
	c.ack(ctx, log, msg.ID)
}

// applyTx запускает транзакцию: dedup + handler. Возвращает committed=true
// если результат был применён в этом вызове (а не дублем).
func (c *Consumer) applyTx(ctx context.Context, env jobs.ResultEnvelope) (committed bool, err error) {
	const op = "redisstream.Consumer.applyTx"

	tx, err := c.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return false, fmt.Errorf("%s: begin: %w", op, err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	inserted, err := c.systemRepo.MarkProcessedTx(ctx, tx, env.JobID, env.Type, env.AgentID)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	if !inserted {
		// дубль — COMMIT пустой, всё равно отметим
		if cErr := tx.Commit(ctx); cErr != nil {
			return false, fmt.Errorf("%s: dup commit: %w", op, cErr)
		}
		return false, nil
	}

	// При status=error агент ничего полезного не вернул — пропускаем handler,
	// но dedup-запись остаётся, чтобы повторно не переобрабатывать.
	if env.Status == jobs.StatusOK {
		if hErr := c.handler(ctx, tx, env); hErr != nil {
			return false, fmt.Errorf("%s: handler: %w", op, hErr)
		}
	}

	if cErr := tx.Commit(ctx); cErr != nil {
		return false, fmt.Errorf("%s: commit: %w", op, cErr)
	}
	return true, nil
}

func (c *Consumer) ack(ctx context.Context, log *slog.Logger, msgID string) {
	if err := c.rdb.XAck(ctx, c.stream, c.group, msgID).Err(); err != nil {
		log.Error("xack", slog.String("error", err.Error()))
	}
}

func decodeEnvelope(msg redis.XMessage) (jobs.ResultEnvelope, error) {
	raw, ok := msg.Values["data"]
	if !ok {
		return jobs.ResultEnvelope{}, errors.New("missing data field")
	}
	s, ok := raw.(string)
	if !ok {
		return jobs.ResultEnvelope{}, errors.New("data is not string")
	}
	var env jobs.ResultEnvelope
	if err := json.Unmarshal([]byte(s), &env); err != nil {
		return jobs.ResultEnvelope{}, fmt.Errorf("decode: %w", err)
	}
	return env, nil
}
