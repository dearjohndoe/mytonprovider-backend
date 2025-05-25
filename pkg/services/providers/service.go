package providers

import (
	"context"
	"log"

	"mytonprovider-backend/pkg/models"
	v1 "mytonprovider-backend/pkg/models/api/v1"
	"mytonprovider-backend/pkg/models/db"
)

type service struct {
	providers providers
	logger    *log.Logger
}

type providers interface {
	GetProvidersByPubkeys(ctx context.Context, pubkeys []string) ([]db.Provider, error)
	GetProviders(ctx context.Context, filters db.ProviderFilters, sort db.ProviderSort, limit, offset int) ([]db.Provider, error)
}

type Providers interface {
	AddProvider(ctx context.Context, provider *db.Provider) (err error)
	GetProviders(ctx context.Context) (providers []*db.Provider, err error)
	UpdateTelemetry(ctx context.Context, telemetry *v1.TelemetryRequest) (err error)
}

func (s *service) AddProvider(ctx context.Context, provider *db.Provider) (err error) {
	return
}

func (s *service) GetProviders(ctx context.Context) (providers []*db.Provider, err error) {
	return
}

func (s *service) UpdateTelemetry(ctx context.Context, telemetry *v1.TelemetryRequest) (err error) {
	if telemetry == nil || telemetry.Storage.Provider.PubKey == "" {
		return models.NewAppError(models.BadRequestErrorCode, "")
	}

	// logic in cache middleware

	return nil
}

func NewService(
	providers providers,
	logger *log.Logger,
) Providers {
	return &service{
		providers: providers,
		logger:    logger,
	}
}
