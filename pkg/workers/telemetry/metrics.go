package telemetry

import (
	"context"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type metricsMiddleware struct {
	reqCount    *prometheus.CounterVec
	reqDuration *prometheus.HistogramVec
	worker      Worker
}

func (m *metricsMiddleware) UpdateTelemetry(ctx context.Context) (interval time.Duration, err error) {
	defer func(s time.Time) {
		labels := []string{
			"UpdateTelemetry", strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.worker.UpdateTelemetry(ctx)
}

func (m *metricsMiddleware) UpdateBenchmarks(ctx context.Context) (interval time.Duration, err error) {
	defer func(s time.Time) {
		labels := []string{
			"UpdateBenchmarks", strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.worker.UpdateBenchmarks(ctx)
}

func NewMetrics(reqCount *prometheus.CounterVec, reqDuration *prometheus.HistogramVec, worker Worker) Worker {
	return &metricsMiddleware{
		reqCount:    reqCount,
		reqDuration: reqDuration,
		worker:      worker,
	}
}
