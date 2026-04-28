package system

import (
	"context"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus"
)

type metricsMiddleware struct {
	reqCount    *prometheus.CounterVec
	reqDuration *prometheus.HistogramVec
	repo        Repository
}

func (m *metricsMiddleware) SetParam(ctx context.Context, key string, value string) (err error) {
	defer func(s time.Time) {
		labels := []string{
			"SetParam", strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.repo.SetParam(ctx, key, value)
}

func (m *metricsMiddleware) GetParam(ctx context.Context, key string) (value string, err error) {
	defer func(s time.Time) {
		labels := []string{
			"GetParam", strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.repo.GetParam(ctx, key)
}

func (m *metricsMiddleware) WithTx(tx pgx.Tx) Repository {
	// metrics middleware прозрачен относительно транзакции:
	// возвращаем чистый repo (без метрик), чтобы tx-bound операции
	// шли мимо обёртки и не путали observability.
	return m.repo.WithTx(tx)
}

func (m *metricsMiddleware) MarkProcessedTx(
	ctx context.Context,
	tx pgx.Tx,
	jobID, jobType, agentID string,
) (inserted bool, err error) {
	defer func(s time.Time) {
		labels := []string{
			"MarkProcessedTx", strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.repo.MarkProcessedTx(ctx, tx, jobID, jobType, agentID)
}

func NewMetrics(reqCount *prometheus.CounterVec, reqDuration *prometheus.HistogramVec, repo Repository) Repository {
	return &metricsMiddleware{
		reqCount:    reqCount,
		reqDuration: reqDuration,
		repo:        repo,
	}
}
