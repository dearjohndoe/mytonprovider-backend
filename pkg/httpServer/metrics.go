package httpServer

import (
	"mytonprovider-backend/pkg/metrics"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
)

func NewMetricsMiddleware(m *metrics.NetMetrics) fiber.Handler {
	return func(ctx *fiber.Ctx) (err error) {
		m.InflightRequests.Inc()
		s := time.Now()
		defer func() {
			m.InflightRequests.Dec()
		}()

		err = ctx.Next()

		routeLabel := "<unmatched>"
		if r := ctx.Route(); r != nil && r.Path != "" {
			routeLabel = r.Path
		}

		labels := []string{
			routeLabel,
			string(ctx.Context().Method()),
			strconv.Itoa(ctx.Context().Response.StatusCode()),
		}

		m.TotalRequests.WithLabelValues(labels...).Inc()
		m.DurationSec.WithLabelValues(labels...).Observe(time.Since(s).Seconds())

		return
	}
}
