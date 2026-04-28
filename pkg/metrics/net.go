package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type NetMetrics struct {
	TotalRequests    *prometheus.CounterVec
	DurationSec      *prometheus.HistogramVec
	InflightRequests prometheus.Gauge
}

type NetMetricsConfig struct {
	Namespace string
	SubSystem string
}

func NewNetMetrics(cfg NetMetricsConfig) *NetMetrics {
	labels := []string{"route", "method", "code"}

	return &NetMetrics{
		TotalRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.SubSystem,
				Name:      "requests_total",
				Help:      "Total number of requests",
			}, labels),

		DurationSec: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.SubSystem,
				Name:      "requests_duration",
				Help:      "Duration of requests",
			}, labels),

		InflightRequests: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.SubSystem,
				Name:      "requests_inflight",
				Help:      "Number of inflight requests",
			},
		),
	}
}
