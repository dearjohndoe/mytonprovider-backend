package providers

import (
	"context"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"mytonprovider-backend/pkg/models/db"
)

type metricsMiddleware struct {
	reqCount    *prometheus.CounterVec
	reqDuration *prometheus.HistogramVec
	repo        Repository
}

func (m *metricsMiddleware) GetProvidersByPubkeys(ctx context.Context, pubkeys []string) (providers []db.ProviderDB, err error) {
	defer func(s time.Time) {
		labels := []string{
			"GetProvidersByPubkeys", strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.repo.GetProvidersByPubkeys(ctx, pubkeys)
}

func (m *metricsMiddleware) GetFilteredProviders(ctx context.Context, filters db.ProviderFilters, sort db.ProviderSort, limit, offset int) (providers []db.ProviderDB, err error) {
	defer func(s time.Time) {
		labels := []string{
			"GetFilteredProviders", strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.repo.GetFilteredProviders(ctx, filters, sort, limit, offset)
}

func (m *metricsMiddleware) UpdateTelemetry(ctx context.Context, telemetry []db.TelemetryUpdate) (err error) {
	defer func(s time.Time) {
		labels := []string{
			"UpdateTelemetry", strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.repo.UpdateTelemetry(ctx, telemetry)
}

func (m *metricsMiddleware) UpdateBenchmarks(ctx context.Context, benchmarks []db.BenchmarkUpdate) (err error) {
	defer func(s time.Time) {
		labels := []string{
			"UpdateBenchmarks", strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.repo.UpdateBenchmarks(ctx, benchmarks)
}

func (m *metricsMiddleware) AddStatuses(ctx context.Context, providers []db.ProviderStatusUpdate) (err error) {
	defer func(s time.Time) {
		labels := []string{
			"AddStatuses", strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.repo.AddStatuses(ctx, providers)
}

func (m *metricsMiddleware) UpdateUptime(ctx context.Context) (err error) {
	defer func(s time.Time) {
		labels := []string{
			"UpdateUptime", strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.repo.UpdateUptime(ctx)
}

func (m *metricsMiddleware) UpdateRating(ctx context.Context) (err error) {
	defer func(s time.Time) {
		labels := []string{
			"UpdateRating", strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.repo.UpdateRating(ctx)
}

func (m *metricsMiddleware) GetAllProvidersPubkeys(ctx context.Context) (pubkeys []string, err error) {
	defer func(s time.Time) {
		labels := []string{
			"GetAllProvidersPubkeys", strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.repo.GetAllProvidersPubkeys(ctx)
}

func (m *metricsMiddleware) GetAllProvidersWallets(ctx context.Context) (wallets []db.ProviderWallet, err error) {
	defer func(s time.Time) {
		labels := []string{
			"GetAllProvidersWallets", strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.repo.GetAllProvidersWallets(ctx)
}

func (m *metricsMiddleware) UpdateProviders(ctx context.Context, providers []db.ProviderUpdate) (err error) {
	defer func(s time.Time) {
		labels := []string{
			"UpdateProviders", strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.repo.UpdateProviders(ctx, providers)
}

func (m *metricsMiddleware) AddProviders(ctx context.Context, providers []db.ProviderCreate) (err error) {
	defer func(s time.Time) {
		labels := []string{
			"AddProviders", strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.repo.AddProviders(ctx, providers)
}

func (m *metricsMiddleware) CleanOldProvidersHistory(ctx context.Context, days int) (removed int, err error) {
	defer func(s time.Time) {
		labels := []string{
			"CleanOldProvidersHistory", strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.repo.CleanOldProvidersHistory(ctx, days)
}

func (m *metricsMiddleware) CleanOldStatusesHistory(ctx context.Context, days int) (removed int, err error) {
	defer func(s time.Time) {
		labels := []string{
			"CleanOldStatusesHistory", strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.repo.CleanOldStatusesHistory(ctx, days)
}

func (m *metricsMiddleware) CleanOldBenchmarksHistory(ctx context.Context, days int) (removed int, err error) {
	defer func(s time.Time) {
		labels := []string{
			"CleanOldBenchmarksHistory",
			strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.repo.CleanOldBenchmarksHistory(ctx, days)
}

func (m *metricsMiddleware) CleanOldTelemetryHistory(ctx context.Context, days int) (removed int, err error) {
	defer func(s time.Time) {
		labels := []string{
			"CleanOldTelemetryHistory", strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.repo.CleanOldTelemetryHistory(ctx, days)
}

func NewMetrics(reqCount *prometheus.CounterVec, reqDuration *prometheus.HistogramVec, repo Repository) Repository {
	return &metricsMiddleware{
		reqCount:    reqCount,
		reqDuration: reqDuration,
		repo:        repo,
	}
}
