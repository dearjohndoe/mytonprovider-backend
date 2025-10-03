package httpServer

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	totalRequests    *prometheus.CounterVec
	durationSec      *prometheus.HistogramVec
	inflightRequests *prometheus.GaugeVec
}

func (m *metrics) metricsMiddleware(ctx *fiber.Ctx) (err error) {
	routeLabel := "<unmatched>"

	if r := ctx.Route(); r != nil && r.Path != "" {
		routeLabel = r.Path
	}

	labels := []string{
		routeLabel,
		string(ctx.Context().Method()),
	}

	m.inflightRequests.WithLabelValues(labels...).Inc()
	defer m.inflightRequests.WithLabelValues(labels...).Dec()

	s := time.Now()

	err = ctx.Next()

	labelsWithCode := []string{
		routeLabel,
		string(ctx.Context().Method()),
		strconv.Itoa(ctx.Response().StatusCode()),
	}

	m.totalRequests.WithLabelValues(labelsWithCode...).Inc()
	m.durationSec.WithLabelValues(labelsWithCode...).Observe(time.Since(s).Seconds())

	return
}

func newMetrics(namespace, subsystem string) *metrics {
	labels := []string{"route", "method", "code"}
	labelsWithoutCode := []string{"route", "method"}

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
	i := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "requests_inflight",
		Help:      "Number of inflight requests",
	}, labelsWithoutCode)

	prometheus.MustRegister(
		t,
		d,
		i,
	)

	return &metrics{
		totalRequests:    t,
		durationSec:      d,
		inflightRequests: i,
	}
}
