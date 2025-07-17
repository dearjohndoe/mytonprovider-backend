package cleaner

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

func (m *metricsMiddleware) CleanupOldData(ctx context.Context) (interval time.Duration, err error) {
	defer func(s time.Time) {
		labels := []string{
			"CleanupOldData", strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.worker.CleanupOldData(ctx)
}

func NewMetrics(reqCount *prometheus.CounterVec, reqDuration *prometheus.HistogramVec, worker Worker) Worker {
	return &metricsMiddleware{
		reqCount:    reqCount,
		reqDuration: reqDuration,
		worker:      worker,
	}
}
