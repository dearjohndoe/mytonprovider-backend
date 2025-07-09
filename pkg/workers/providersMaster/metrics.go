package providersmaster

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

func (m *metricsMiddleware) CollectNewProviders(ctx context.Context) (interval time.Duration, err error) {
	defer func(s time.Time) {
		labels := []string{
			"CollectNewProviders", strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.worker.CollectNewProviders(ctx)
}

func (m *metricsMiddleware) UpdateKnownProviders(ctx context.Context) (interval time.Duration, err error) {
	defer func(s time.Time) {
		labels := []string{
			"UpdateKnownProviders", strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.worker.UpdateKnownProviders(ctx)
}

func (m *metricsMiddleware) CollectProvidersNewStorageContracts(ctx context.Context) (interval time.Duration, err error) {
	defer func(s time.Time) {
		labels := []string{
			"CollectProvidersNewStorageContracts", strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.worker.CollectProvidersNewStorageContracts(ctx)
}

func (m *metricsMiddleware) UpdateUptime(ctx context.Context) (interval time.Duration, err error) {
	defer func(s time.Time) {
		labels := []string{
			"UpdateUptime", strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.worker.UpdateUptime(ctx)
}

func (m *metricsMiddleware) UpdateRating(ctx context.Context) (interval time.Duration, err error) {
	defer func(s time.Time) {
		labels := []string{
			"UpdateRating", strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.worker.UpdateRating(ctx)
}

func NewMetrics(reqCount *prometheus.CounterVec, reqDuration *prometheus.HistogramVec, worker Worker) Worker {
	return &metricsMiddleware{
		reqCount:    reqCount,
		reqDuration: reqDuration,
		worker:      worker,
	}
}
