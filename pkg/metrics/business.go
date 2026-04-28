package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type BusinessMetrics struct {
	DBRequests              *prometheus.CounterVec
	DBRequestsDuration      *prometheus.HistogramVec
	WorkersRequests         *prometheus.CounterVec
	WorkersRequestsDuration *prometheus.HistogramVec
	ProvidersNetLoad        *prometheus.GaugeVec
}

type BusinessMetricsConfig struct {
	Namespace          string
	ServerSubSystem    string
	WorkerSubSystem    string
	DBSubSystem        string
	ProvidersSubSystem string
}

func NewBusinessMetrics(cfg BusinessMetricsConfig) *BusinessMetrics {
	return &BusinessMetrics{
		DBRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.DBSubSystem,
				Name:      "db_requests_count",
				Help:      "Db requests count",
			},
			[]string{"method", "error"},
		),

		DBRequestsDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.DBSubSystem,
				Name:      "db_requests_duration",
				Help:      "Db requests duration",
			},
			[]string{"method", "error"},
		),

		WorkersRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.WorkerSubSystem,
				Name:      "workers_requests_count",
				Help:      "Workers requests count",
			},
			[]string{"method", "error"},
		),

		WorkersRequestsDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.WorkerSubSystem,
				Name:      "workers_requests_duration",
				Help:      "Workers requests duration",
			},
			[]string{"method", "error"},
		),

		ProvidersNetLoad: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.ProvidersSubSystem,
				Name:      "providers_net_load",
				Help:      "Providers network load",
			},
			[]string{"provider_pubkey", "type"},
		),
	}
}
