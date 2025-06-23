package httpServer

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	totalRequests *prometheus.CounterVec
	durationSec   *prometheus.HistogramVec
}

func (m *metrics) metricsMiddleware(ctx *fiber.Ctx) (err error) {
	s := time.Now()

	err = ctx.Next()

	labels := []string{
		ctx.Context().URI().String(),
		string(ctx.Context().Method()),
		strconv.Itoa(ctx.Response().StatusCode()),
	}

	m.totalRequests.WithLabelValues(labels...).Inc()
	m.durationSec.WithLabelValues(labels...).Observe(time.Since(s).Seconds())

	return
}

func newMetrics(namespace, subsystem string) *metrics {
	labels := []string{"route", "method", "code"}

	t := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "requests_total",
		Help:      "Total number of requests",
	}, labels)
	d := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "requests_duration",
		Help:      "Duration of requests",
	}, labels)

	prometheus.MustRegister(
		t,
		d,
	)

	return &metrics{
		totalRequests: t,
		durationSec:   d,
	}
}
