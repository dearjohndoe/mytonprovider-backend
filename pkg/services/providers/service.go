package providers

import (
	"context"
	"log"
	"time"

	"mytonprovider-backend/pkg/models"
	v1 "mytonprovider-backend/pkg/models/api/v1"
	"mytonprovider-backend/pkg/models/db"
)

const (
	maxProvidersLimit = 1000
)

type service struct {
	providers providers
	logger    *log.Logger
}

type providers interface {
	GetProvidersByPubkeys(ctx context.Context, pubkeys []string) ([]db.ProviderDB, error)
	GetProviders(ctx context.Context, filters db.ProviderFilters, sort db.ProviderSort, limit, offset int) ([]db.ProviderDB, error)
}

type Providers interface {
	SearchProviders(ctx context.Context, req v1.SearchProvidersRequest) (providers []v1.Provider, err error)
	GetLatestTelemetry(ctx context.Context) (providers []v1.TelemetryRequest, err error)
	UpdateTelemetry(ctx context.Context, telemetry *v1.TelemetryRequest) (err error)
}

func (s *service) SearchProviders(ctx context.Context, req v1.SearchProvidersRequest) (providers []v1.Provider, err error) {
	if len(req.Exact) > 0 {
		if len(req.Exact) > maxProvidersLimit {
			s.logger.Printf("Too many pubkeys in request: %d, max allowed: %d", len(req.Exact), maxProvidersLimit)
			err = models.NewAppError(models.BadRequestErrorCode, "too many pubkeys in request")
			return
		}

		p, dbErr := s.providers.GetProvidersByPubkeys(ctx, req.Exact)
		if dbErr != nil {
			s.logger.Printf("Failed to get providers by pubkeys: %v", dbErr)
			err = models.NewAppError(models.InternalServerErrorCode, "")
			return
		}

		providers = convertDBProvidersToAPI(p)

		return
	}

	filters, sort, limit, offset := buildProviderQueryParams(req)

	p, dbErr := s.providers.GetProviders(ctx, filters, sort, limit, offset)
	if dbErr != nil {
		s.logger.Printf("Failed to get providers: %v", dbErr)
		err = models.NewAppError(models.InternalServerErrorCode, "")
		return
	}

	providers = convertDBProvidersToAPI(p)

	return
}

func (s *service) GetLatestTelemetry(ctx context.Context) (providers []v1.TelemetryRequest, err error) {
	// logic in cache middleware

	return
}

func (s *service) UpdateTelemetry(ctx context.Context, telemetry *v1.TelemetryRequest) (err error) {
	if telemetry == nil || telemetry.Storage.Provider.PubKey == "" {
		return models.NewAppError(models.BadRequestErrorCode, "")
	}

	// logic in cache middleware

	return nil
}

func convertDBProvidersToAPI(providersDB []db.ProviderDB) []v1.Provider {
	providers := make([]v1.Provider, 0, len(providersDB))
	for _, provider := range providersDB {
		workingTime := uint64(time.Now().Unix()) - provider.RegTime

		providers = append(providers, v1.Provider{
			PubKey:          provider.PubKey,
			UpTime:          provider.UpTime,
			WorkingTime:     workingTime,
			Rating:          provider.Rating,
			MaxSpan:         provider.MaxSpan,
			Price:           provider.Price,
			MinSpan:         provider.MinSpan,
			MaxBagSizeBytes: provider.MaxBagSizeBytes,
			RegTime:         provider.RegTime,
			IsSendTelemetry: provider.IsSendTelemetry,
			Telemetry: v1.Telemetry{
				StorageGitHash:          provider.Telemetry.StorageGitHash,
				ProviderGitHash:         provider.Telemetry.ProviderGitHash,
				TotalProviderSpace:      provider.Telemetry.TotalProviderSpace,
				FreeProviderSpace:       provider.Telemetry.FreeProviderSpace,
				CPUName:                 provider.Telemetry.CPUName,
				CPUNumber:               provider.Telemetry.CPUNumber,
				CPUIsVirtual:            provider.Telemetry.CPUIsVirtual,
				TotalRAM:                provider.Telemetry.TotalRAM,
				FreeRAM:                 provider.Telemetry.FreeRAM,
				BenchmarkDiskReadSpeed:  provider.Telemetry.BenchmarkDiskReadSpeed,
				BenchmarkDiskWriteSpeed: provider.Telemetry.BenchmarkDiskWriteSpeed,
				BenchmarkRocksOps:       provider.Telemetry.BenchmarkRocksOps,
				SpeedtestDownloadSpeed:  provider.Telemetry.SpeedtestDownloadSpeed,
				SpeedtestUploadSpeed:    provider.Telemetry.SpeedtestUploadSpeed,
				SpeedtestPing:           provider.Telemetry.SpeedtestPing,
				Country:                 provider.Telemetry.Country,
				ISP:                     provider.Telemetry.ISP,
			},
		})
	}
	return providers
}

func buildProviderQueryParams(req v1.SearchProvidersRequest) (db.ProviderFilters, db.ProviderSort, int, int) {
	filters := db.ProviderFilters{
		RatingGt:                  req.Filters.RatingGt,
		RatingLt:                  req.Filters.RatingLt,
		RegTimeGtDays:             req.Filters.RegTimeGtDays,
		RegTimeLtDays:             req.Filters.RegTimeLtDays,
		UpTimeGtPercent:           req.Filters.UpTimeGtPercent,
		UpTimeLtPercent:           req.Filters.UpTimeLtPercent,
		WorkingTimeGtSec:          req.Filters.WorkingTimeGtSec,
		WorkingTimeLtSec:          req.Filters.WorkingTimeLtSec,
		PriceGt:                   req.Filters.PriceGt,
		PriceLt:                   req.Filters.PriceLt,
		MinSpanGt:                 req.Filters.MinSpanGt,
		MinSpanLt:                 req.Filters.MinSpanLt,
		MaxSpanGt:                 req.Filters.MaxSpanGt,
		MaxSpanLt:                 req.Filters.MaxSpanLt,
		MaxBagSizeBytesGt:         req.Filters.MaxBagSizeBytesGt,
		MaxBagSizeBytesLt:         req.Filters.MaxBagSizeBytesLt,
		IsSendTelemetry:           req.Filters.IsSendTelemetry,
		TotalProviderSpaceGt:      req.Filters.TotalProviderSpaceGt,
		TotalProviderSpaceLt:      req.Filters.TotalProviderSpaceLt,
		FreeProviderSpaceGt:       req.Filters.FreeProviderSpaceGt,
		FreeProviderSpaceLt:       req.Filters.FreeProviderSpaceLt,
		StorageGitHash:            req.Filters.StorageGitHash,
		ProviderGitHash:           req.Filters.ProviderGitHash,
		CPUNumberGt:               req.Filters.CPUNumberGt,
		CPUNumberLt:               req.Filters.CPUNumberLt,
		CPUName:                   req.Filters.CPUName,
		CPUIsVirtual:              req.Filters.CPUIsVirtual,
		TotalRamGt:                req.Filters.TotalRamGt,
		TotalRamLt:                req.Filters.TotalRamLt,
		FreeRamGt:                 req.Filters.FreeRamGt,
		FreeRamLt:                 req.Filters.FreeRamLt,
		BenchmarkDiskReadSpeedGt:  req.Filters.BenchmarkDiskReadSpeedGt,
		BenchmarkDiskReadSpeedLt:  req.Filters.BenchmarkDiskReadSpeedLt,
		BenchmarkDiskWriteSpeedGt: req.Filters.BenchmarkDiskWriteSpeedGt,
		BenchmarkDiskWriteSpeedLt: req.Filters.BenchmarkDiskWriteSpeedLt,
		BenchmarkRocksOpsGt:       req.Filters.BenchmarkRocksOpsGt,
		BenchmarkRocksOpsLt:       req.Filters.BenchmarkRocksOpsLt,
		SpeedtestDownloadSpeedGt:  req.Filters.SpeedtestDownloadSpeedGt,
		SpeedtestDownloadSpeedLt:  req.Filters.SpeedtestDownloadSpeedLt,
		SpeedtestUploadSpeedGt:    req.Filters.SpeedtestUploadSpeedGt,
		SpeedtestUploadSpeedLt:    req.Filters.SpeedtestUploadSpeedLt,
		SpeedtestPingGt:           req.Filters.SpeedtestPingGt,
		SpeedtestPingLt:           req.Filters.SpeedtestPingLt,
		Country:                   req.Filters.Country,
		ISP:                       req.Filters.ISP,
	}

	sort := db.ProviderSort{
		Column: req.Sort.Column,
		Order:  req.Sort.Order,
	}

	limit := req.Limit
	if limit <= 0 || limit > maxProvidersLimit {
		limit = maxProvidersLimit
	}
	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	return filters, sort, limit, offset
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
