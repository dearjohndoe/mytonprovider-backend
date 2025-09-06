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

func (m *metricsMiddleware) GetFiltersRange(ctx context.Context) (filtersRange db.FiltersRange, err error) {
	defer func(s time.Time) {
		labels := []string{
			"GetFiltersRange", strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.repo.GetFiltersRange(ctx)
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

func (m *metricsMiddleware) GetStorageContractsChecks(ctx context.Context, contracts []string) (resp []db.ContractCheck, err error) {
	defer func(s time.Time) {
		labels := []string{
			"GetStorageContractsChecks", strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.repo.GetStorageContractsChecks(ctx, contracts)
}

func (m *metricsMiddleware) UpdateContractProofsChecks(ctx context.Context, contractsProofs []db.ContractProofsCheck) (err error) {
	defer func(s time.Time) {
		labels := []string{
			"UpdateContractProofsChecks", strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.repo.UpdateContractProofsChecks(ctx, contractsProofs)
}

func (m *metricsMiddleware) UpdateProvidersIPs(ctx context.Context, ips []db.ProviderIP) (err error) {
	defer func(s time.Time) {
		labels := []string{
			"UpdateProvidersIPs", strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.repo.UpdateProvidersIPs(ctx, ips)
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

func (m *metricsMiddleware) UpdateProvidersLT(ctx context.Context, providers []db.ProviderWalletLT) (err error) {
	defer func(s time.Time) {
		labels := []string{
			"UpdateProvidersLT", strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.repo.UpdateProvidersLT(ctx, providers)
}

func (m *metricsMiddleware) AddStorageContracts(ctx context.Context, contracts []db.StorageContract) (err error) {
	defer func(s time.Time) {
		labels := []string{
			"AddStorageContracts", strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.repo.AddStorageContracts(ctx, contracts)
}

func (m *metricsMiddleware) UpdateStatuses(ctx context.Context) (err error) {
	defer func(s time.Time) {
		labels := []string{
			"UpdateStatuses", strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.repo.UpdateStatuses(ctx)
}

func (m *metricsMiddleware) GetStorageContracts(ctx context.Context) (contracts []db.ContractToProviderRelation, err error) {
	defer func(s time.Time) {
		labels := []string{
			"GetStorageContracts", strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.repo.GetStorageContracts(ctx)
}

func (m *metricsMiddleware) UpdateRejectedStorageContracts(ctx context.Context, storageContracts []db.ContractToProviderRelation) (err error) {
	defer func(s time.Time) {
		labels := []string{
			"UpdateRejectedStorageContracts", strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.repo.UpdateRejectedStorageContracts(ctx, storageContracts)
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

func (m *metricsMiddleware) GetProvidersIPs(ctx context.Context) (ips []db.ProviderIP, err error) {
	defer func(s time.Time) {
		labels := []string{
			"GetProvidersIPs", strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.repo.GetProvidersIPs(ctx)
}

func (m *metricsMiddleware) UpdateProvidersIPInfo(ctx context.Context, ips []db.ProviderIPInfo) (err error) {
	defer func(s time.Time) {
		labels := []string{
			"UpdateProvidersIPInfo", strconv.FormatBool(err != nil),
		}
		m.reqCount.WithLabelValues(labels...).Add(1)
		m.reqDuration.WithLabelValues(labels...).Observe(time.Since(s).Seconds())
	}(time.Now())
	return m.repo.UpdateProvidersIPInfo(ctx, ips)
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
