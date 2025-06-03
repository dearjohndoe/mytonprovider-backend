package providers

import (
	"context"
	"strings"
	"time"

	"mytonprovider-backend/pkg/cache"
	v1 "mytonprovider-backend/pkg/models/api/v1"
)

type cacheMiddleware struct {
	svc    Providers
	buffer *cache.SimpleCache
	lates  *cache.SimpleCache
}

func (c *cacheMiddleware) SearchProviders(ctx context.Context, req v1.SearchProvidersRequest) (providers []v1.Provider, err error) {
	return c.svc.SearchProviders(ctx, req)
}

func (c *cacheMiddleware) GetLatestTelemetry(ctx context.Context) (providers []v1.TelemetryRequest, err error) {
	data := c.lates.GetAll()
	if len(data) == 0 {
		return
	}

	providers = make([]v1.TelemetryRequest, 0, len(data))
	for _, v := range data {
		if telemetry, ok := v.(*v1.TelemetryRequest); ok && telemetry != nil {
			providers = append(providers, *telemetry)
		}
	}

	return
}

func (c *cacheMiddleware) UpdateTelemetry(ctx context.Context, telemetry *v1.TelemetryRequest) (err error) {
	err = c.svc.UpdateTelemetry(ctx, telemetry)
	if err != nil {
		return
	}

	c.buffer.Set(strings.ToLower(telemetry.Storage.Provider.PubKey), telemetry)
	c.lates.Set(strings.ToLower(telemetry.Storage.Provider.PubKey), telemetry)

	return
}

func NewCacheMiddleware(svc Providers, telemetry *cache.SimpleCache) Providers {
	latest := cache.NewSimpleCache(2 * time.Minute)
	return &cacheMiddleware{
		svc:    svc,
		buffer: telemetry,
		lates:  latest,
	}
}
