package providers

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"mytonprovider-backend/pkg/cache"
	v1 "mytonprovider-backend/pkg/models/api/v1"
	"mytonprovider-backend/pkg/utils"
)

const (
	filtersRangeKey = "filtersRange"
)

type cacheMiddleware struct {
	svc                   Providers
	telemetryBuffer       *cache.SimpleCache
	benchmarksBuffer      *cache.SimpleCache
	latestTelemetryBuffer *cache.SimpleCache
	cache                 *cache.SimpleCache
}

func (c *cacheMiddleware) SearchProviders(ctx context.Context, req v1.SearchProvidersRequest) (providers []v1.Provider, err error) {
	return c.svc.SearchProviders(ctx, req)
}

func (c *cacheMiddleware) GetFiltersRange(ctx context.Context) (filtersRange v1.FiltersRangeResp, err error) {
	v, ok := c.cache.Get(filtersRangeKey)
	if !ok {
		return c.actualFiltersRange(ctx)
	}

	filtersRange, ok = v.(v1.FiltersRangeResp)
	if ok {
		return
	}

	return c.actualFiltersRange(ctx)
}

func (c *cacheMiddleware) GetLatestTelemetry(ctx context.Context) (providers []interface{}, err error) {
	data := c.latestTelemetryBuffer.GetAll()
	if len(data) == 0 {
		return
	}

	providers = make([]interface{}, 0, len(data))
	for _, v := range data {
		if telemetry, ok := v.([]byte); ok {
			var telemetryCopy interface{}
			err := json.Unmarshal(telemetry, &telemetryCopy)
			if err == nil {
				providers = append(providers, telemetryCopy)
			}
		}
	}

	return
}

func (c *cacheMiddleware) UpdateTelemetry(ctx context.Context, telemetry v1.TelemetryRequest, rawBody []byte) (err error) {
	err = c.svc.UpdateTelemetry(ctx, telemetry, rawBody)
	if err != nil {
		return
	}

	telemetryCopy, copyErr := utils.DeepCopy(telemetry)
	if copyErr != nil {
		telemetryCopy = telemetry
	}

	key := strings.ToLower(telemetry.Storage.Provider.PubKey)
	c.telemetryBuffer.Set(key, telemetryCopy)
	c.latestTelemetryBuffer.Set(key, rawBody)

	return
}

func (c *cacheMiddleware) UpdateBenchmarks(ctx context.Context, benchmark v1.BenchmarksRequest) (err error) {
	err = c.svc.UpdateBenchmarks(ctx, benchmark)
	if err != nil {
		return
	}

	benchmarkCopy, copyErr := utils.DeepCopy(benchmark)
	if copyErr != nil {
		benchmarkCopy = benchmark
	}
	c.benchmarksBuffer.Set(strings.ToLower(benchmark.PubKey), benchmarkCopy)

	return
}

func (c *cacheMiddleware) GetStorageContractsChecks(ctx context.Context, req v1.ContractsStatusesRequest) ([]v1.ContractCheck, error) {
	return c.svc.GetStorageContractsChecks(ctx, req)
}

func (c *cacheMiddleware) actualFiltersRange(ctx context.Context) (filtersRange v1.FiltersRangeResp, err error) {
	filtersRange, err = c.svc.GetFiltersRange(ctx)
	if err != nil {
		return
	}

	c.cache.Set(filtersRangeKey, filtersRange)
	return
}

func NewCacheMiddleware(
	svc Providers,
	telemetry *cache.SimpleCache,
	benchmarks *cache.SimpleCache,
) Providers {
	latest := cache.NewSimpleCache(2 * time.Minute)
	return &cacheMiddleware{
		svc:                   svc,
		telemetryBuffer:       telemetry,
		benchmarksBuffer:      benchmarks,
		cache:                 cache.NewSimpleCache(1 * time.Minute),
		latestTelemetryBuffer: latest,
	}
}
