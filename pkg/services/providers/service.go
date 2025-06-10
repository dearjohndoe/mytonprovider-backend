package providers

import (
	"context"
	"log/slog"
	"time"

	"mytonprovider-backend/pkg/constants"
	"mytonprovider-backend/pkg/models"
	v1 "mytonprovider-backend/pkg/models/api/v1"
	"mytonprovider-backend/pkg/models/db"
)

const (
	maxProvidersLimit = 1000
)

type service struct {
	providers providers
	logger    *slog.Logger
}

type providers interface {
	GetProvidersByPubkeys(ctx context.Context, pubkeys []string) ([]db.ProviderDB, error)
	GetFilteredProviders(ctx context.Context, filters db.ProviderFilters, sort db.ProviderSort, limit, offset int) ([]db.ProviderDB, error)
}

type Providers interface {
	SearchProviders(ctx context.Context, req v1.SearchProvidersRequest) (providers []v1.Provider, err error)
	GetLatestTelemetry(ctx context.Context) (providers []v1.TelemetryRequest, err error)
	UpdateTelemetry(ctx context.Context, telemetry *v1.TelemetryRequest) (err error)
}

func (s *service) SearchProviders(ctx context.Context, req v1.SearchProvidersRequest) (providers []v1.Provider, err error) {
	log := s.logger.With(slog.String("method", "SearchProviders"))

	if len(req.Exact) > 0 {
		providers, err = s.getExactProviders(ctx, req.Exact, log)
		return
	}

	providers, err = s.getFilteredProviders(ctx, req, log)

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

func (s *service) getExactProviders(ctx context.Context, pubkeys []string, log *slog.Logger) (providers []v1.Provider, err error) {
	if len(pubkeys) > maxProvidersLimit {
		log.Error("too many pubkeys in request")
		err = models.NewAppError(models.BadRequestErrorCode, "too many pubkeys in request")
		return
	}

	p, dbErr := s.providers.GetProvidersByPubkeys(ctx, pubkeys)
	if dbErr != nil {
		log.Error("failed to get providers by pubkeys", slog.Any("pubkeys", pubkeys), slog.String("error", dbErr.Error()))
		err = models.NewAppError(models.InternalServerErrorCode, "")
		return
	}

	providers = convertDBProvidersToAPI(p)

	return
}

func (s *service) getFilteredProviders(ctx context.Context, req v1.SearchProvidersRequest, log *slog.Logger) (providers []v1.Provider, err error) {
	filters, sort, limit, offset := buildProviderQueryParams(req)

	p, dbErr := s.providers.GetFilteredProviders(ctx, filters, sort, limit, offset)
	if dbErr != nil {
		log.Error("failed to get providers", slog.String("error", dbErr.Error()))
		err = models.NewAppError(models.InternalServerErrorCode, "")
		return
	}

	providers = convertDBProvidersToAPI(p)

	return
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
				UsedProviderSpace:       provider.Telemetry.UsedProviderSpace,
				CPUName:                 provider.Telemetry.CPUName,
				CPUNumber:               provider.Telemetry.CPUNumber,
				CPUIsVirtual:            provider.Telemetry.CPUIsVirtual,
				TotalRAM:                provider.Telemetry.TotalRAM,
				UsageRAM:                provider.Telemetry.UsageRAM,
				UsageRAMPercent:         provider.Telemetry.UsageRAMPercent,
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
		RegTimeDaysGt:             req.Filters.RegTimeDaysGt,
		RegTimeDaysLt:             req.Filters.RegTimeDaysLt,
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
		UsedProviderSpaceGt:       req.Filters.UsedProviderSpaceGt,
		UsedProviderSpaceLt:       req.Filters.UsedProviderSpaceLt,
		StorageGitHash:            req.Filters.StorageGitHash,
		ProviderGitHash:           req.Filters.ProviderGitHash,
		CPUNumberGt:               req.Filters.CPUNumberGt,
		CPUNumberLt:               req.Filters.CPUNumberLt,
		CPUName:                   req.Filters.CPUName,
		CPUIsVirtual:              req.Filters.CPUIsVirtual,
		TotalRamGt:                req.Filters.TotalRamGt,
		TotalRamLt:                req.Filters.TotalRamLt,
		UsageRamPercentGt:         req.Filters.UsageRamPercentGt,
		UsageRamPercentLt:         req.Filters.UsageRamPercentLt,
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

	sortColumn := constants.RatingColumn
	if v, ok := constants.SortingMap[req.Sort.Column]; ok {
		sortColumn = v
	}
	order := constants.Asc
	if v, ok := constants.OrderMap[req.Sort.Order]; ok {
		order = v
	}
	sort := db.ProviderSort{
		Column: sortColumn,
		Order:  order,
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
	logger *slog.Logger,
) Providers {
	return &service{
		providers: providers,
		logger:    logger,
	}
}
