package providers

import (
	"context"
	"log"

	"github.com/patrickmn/go-cache"

	"mytonprovider-backend/pkg/models"
)

type service struct {
	cache  *cache.Cache
	logger *log.Logger
}

type Providers interface {
	AddProvider(ctx context.Context, provider *models.Provider) error
	GetProviders(ctx context.Context) ([]*models.Provider, error)
}

func (s *service) AddProvider(ctx context.Context, provider *models.Provider) (err error) {
	return
}

func (s *service) GetProviders(ctx context.Context) (providers []*models.Provider, err error) {
	return
}

func NewService(
	cacheTelemetry *cache.Cache,
	logger *log.Logger,
) Providers {
	return &service{
		cache:  cacheTelemetry,
		logger: logger,
	}
}
