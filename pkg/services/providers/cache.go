package providers

import (
	"context"

	"mytonprovider-backend/pkg/cache"
	v1 "mytonprovider-backend/pkg/models/api/v1"
	"mytonprovider-backend/pkg/models/db"
)

type cacheMiddleware struct {
	svc       Providers
	telemetry *cache.SimpleCache
}

func (c *cacheMiddleware) AddProvider(ctx context.Context, provider *db.Provider) (err error) {
	return c.svc.AddProvider(ctx, provider)
}

func (c *cacheMiddleware) GetProviders(ctx context.Context) (providers []*db.Provider, err error) {
	return c.svc.GetProviders(ctx)
}

func (c *cacheMiddleware) UpdateTelemetry(ctx context.Context, telemetry *v1.TelemetryRequest) (err error) {
	err = c.svc.UpdateTelemetry(ctx, telemetry)
	if err != nil {
		return
	}

	c.telemetry.Set(telemetry.Storage.Provider.PubKey, telemetry)

	return
}

func NewCacheMiddleware(svc Providers, telemetry *cache.SimpleCache) Providers {
	return &cacheMiddleware{
		svc:       svc,
		telemetry: telemetry,
	}
}
