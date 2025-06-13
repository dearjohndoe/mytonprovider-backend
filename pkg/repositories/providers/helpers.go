package providers

import (
	"fmt"

	"mytonprovider-backend/pkg/models/db"
)

const (
	providersQuerySelect = `
		SELECT 
			p.public_key,
			p.uptime * 100 as uptime,
			p.rating,
			p.max_span,
			GREATEST(p.rate_per_mb_per_day * 1024 * 200 * 30) as price,
			p.min_span,
			0,                  -- p.max_bag_size_bytes ???
			p.registered_at,
			coalesce(p.is_send_telemetry, false) as is_send_telemetry,
			t.storage_git_hash,
			t.provider_git_hash,
			t.total_provider_space,
			t.used_provider_space,
			t.cpu_name,
			t.cpu_number,
			t.cpu_is_virtual,
			t.total_ram,
			t.usage_ram,
			t.ram_usage_percent,
			b.qd64_disk_read_speed,
			b.qd64_disk_write_speed,
			b.speedtest_download,
			b.speedtest_upload,
			b.speedtest_ping,
			b.country,
			b.isp
		FROM providers.providers p
			LEFT JOIN providers.telemetry t ON p.public_key = t.public_key
			LEFT JOIN providers.benchmarks b ON p.public_key = b.public_key
		WHERE p.is_initialized AND p.rating is not null AND p.uptime IS NOT NULL
			%s
		ORDER BY %s
		LIMIT $1
		OFFSET $2`
)

func sortToCondition(sort db.ProviderSort) (condition string) {
	if sort.Column == "" {
		condition = "p.rating "
	} else {
		condition = sort.Column + " "
	}

	if sort.Order == "" {
		condition += "DESC"
	} else {
		if sort.Order == "ASC" {
			condition += "ASC"
		} else {
			condition += "DESC"
		}
	}

	return
}

func filtersToCondition(filters db.ProviderFilters, args []any) (condition string, resArgs []any) {
	resArgs = args

	if filters.RatingGt != nil {
		condition += fmt.Sprintf(" AND p.rating >= %f", *filters.RatingGt)
	}
	if filters.RatingLt != nil {
		condition += fmt.Sprintf(" AND p.rating <= %f", *filters.RatingLt)
	}
	if filters.RegTimeDaysGt != nil {
		condition += fmt.Sprintf(" AND p.registered_at <= NOW() - INTERVAL '%d days'", *filters.RegTimeDaysGt)
	}
	if filters.RegTimeDaysLt != nil {
		condition += fmt.Sprintf(" AND p.registered_at >= NOW() - INTERVAL '%d days'", *filters.RegTimeDaysLt)
	}
	if filters.UpTimeGtPercent != nil {
		uptime := *filters.UpTimeGtPercent / 100.0
		condition += fmt.Sprintf(" AND p.uptime >= %f", uptime)
	}
	if filters.UpTimeLtPercent != nil {
		uptime := *filters.UpTimeLtPercent / 100.0
		condition += fmt.Sprintf(" AND p.uptime <= %f", uptime)
	}
	if filters.WorkingTimeGtSec != nil {
		condition += fmt.Sprintf(" AND p.working_time >= %d", *filters.WorkingTimeGtSec)
	}
	if filters.WorkingTimeLtSec != nil {
		condition += fmt.Sprintf(" AND p.working_time <= %d", *filters.WorkingTimeLtSec)
	}
	if filters.PriceGt != nil {
		condition += fmt.Sprintf(" AND p.rate_per_mb_per_day >= %f", *filters.PriceGt)
	}
	if filters.PriceLt != nil {
		condition += fmt.Sprintf(" AND p.rate_per_mb_per_day <= %f", *filters.PriceLt)
	}
	if filters.MinSpanGt != nil {
		condition += fmt.Sprintf(" AND p.min_span >= %d", *filters.MinSpanGt)
	}
	if filters.MinSpanLt != nil {
		condition += fmt.Sprintf(" AND p.min_span <= %d", *filters.MinSpanLt)
	}
	if filters.MaxSpanGt != nil {
		condition += fmt.Sprintf(" AND p.max_span >= %d", *filters.MaxSpanGt)
	}
	if filters.MaxSpanLt != nil {
		condition += fmt.Sprintf(" AND p.max_span <= %d", *filters.MaxSpanLt)
	}
	// if filters.MaxBagSizeBytesGt != nil {
	// 	condition += fmt.Sprintf(" AND p.max_bag_size_bytes >= %d", *filters.MaxBagSizeBytesGt)
	// }
	// if filters.MaxBagSizeBytesLt != nil {
	// 	condition += fmt.Sprintf(" AND p.max_bag_size_bytes <= %d", *filters.MaxBagSizeBytesLt)
	// }
	if filters.IsSendTelemetry != nil {
		if *filters.IsSendTelemetry {
			condition += " AND p.is_send_telemetry"
		} else {
			condition += " AND (p.is_send_telemetry IS NULL OR NOT p.is_send_telemetry)"
		}
	}
	if filters.TotalProviderSpaceGt != nil {
		condition += fmt.Sprintf(" AND t.total_provider_space >= %f", *filters.TotalProviderSpaceGt)
	}
	if filters.TotalProviderSpaceLt != nil {
		condition += fmt.Sprintf(" AND t.total_provider_space <= %f", *filters.TotalProviderSpaceLt)
	}
	if filters.UsedProviderSpaceGt != nil {
		condition += fmt.Sprintf(" AND t.used_provider_space >= %f", *filters.UsedProviderSpaceGt)
	}
	if filters.UsedProviderSpaceLt != nil {
		condition += fmt.Sprintf(" AND t.used_provider_space <= %f", *filters.UsedProviderSpaceLt)
	}
	if filters.StorageGitHash != nil && len(*filters.StorageGitHash) == 7 {
		resArgs = append(resArgs, *filters.StorageGitHash)
		condition += fmt.Sprintf(" AND t.storage_git_hash = $%d", len(resArgs))
	}
	if filters.ProviderGitHash != nil && len(*filters.ProviderGitHash) == 7 {
		resArgs = append(resArgs, *filters.ProviderGitHash)
		condition += fmt.Sprintf(" AND t.provider_git_hash = $%d", len(resArgs))
	}
	if filters.CPUNumberGt != nil {
		condition += fmt.Sprintf(" AND t.cpu_number >= %d", *filters.CPUNumberGt)
	}
	if filters.CPUNumberLt != nil {
		condition += fmt.Sprintf(" AND t.cpu_number <= %d", *filters.CPUNumberLt)
	}
	if filters.CPUName != nil && len(*filters.CPUName) >= 0 {
		resArgs = append(resArgs, "%"+*filters.CPUName+"%")
		condition += fmt.Sprintf(" AND t.cpu_name ILIKE $%d", len(resArgs))
	}
	if filters.CPUIsVirtual != nil {
		if *filters.CPUIsVirtual {
			condition += " AND t.cpu_is_virtual"
		} else {
			condition += " AND (t.cpu_is_virtual IS NULL OR NOT t.cpu_is_virtual)"
		}
	}
	if filters.TotalRamGt != nil {
		condition += fmt.Sprintf(" AND t.total_ram >= %f", *filters.TotalRamGt)
	}
	if filters.TotalRamLt != nil {
		condition += fmt.Sprintf(" AND t.total_ram <= %f", *filters.TotalRamLt)
	}
	if filters.UsageRamPercentGt != nil {
		condition += fmt.Sprintf(" AND t.ram_usage_percent >= %f", *filters.UsageRamPercentGt)
	}
	if filters.UsageRamPercentLt != nil {
		condition += fmt.Sprintf(" AND t.ram_usage_percent <= %f", *filters.UsageRamPercentLt)
	}
	if filters.BenchmarkDiskReadSpeedGt != nil {
		condition += fmt.Sprintf(" AND t.benchmark_disk_read_speed >= %f", *filters.BenchmarkDiskReadSpeedGt)
	}
	if filters.BenchmarkDiskReadSpeedLt != nil {
		condition += fmt.Sprintf(" AND t.benchmark_disk_read_speed <= %f", *filters.BenchmarkDiskReadSpeedLt)
	}
	if filters.BenchmarkDiskWriteSpeedGt != nil {
		condition += fmt.Sprintf(" AND t.benchmark_disk_write_speed >= %f", *filters.BenchmarkDiskWriteSpeedGt)
	}
	if filters.BenchmarkDiskWriteSpeedLt != nil {
		condition += fmt.Sprintf(" AND t.benchmark_disk_write_speed <= %f", *filters.BenchmarkDiskWriteSpeedLt)
	}
	if filters.SpeedtestDownloadSpeedGt != nil {
		condition += fmt.Sprintf(" AND t.speedtest_download_speed >= %f", *filters.SpeedtestDownloadSpeedGt)
	}
	if filters.SpeedtestDownloadSpeedLt != nil {
		condition += fmt.Sprintf(" AND t.speedtest_download_speed <= %f", *filters.SpeedtestDownloadSpeedLt)
	}
	if filters.SpeedtestUploadSpeedGt != nil {
		condition += fmt.Sprintf(" AND t.speedtest_upload_speed >= %f", *filters.SpeedtestUploadSpeedGt)
	}
	if filters.SpeedtestUploadSpeedLt != nil {
		condition += fmt.Sprintf(" AND t.speedtest_upload_speed <= %f", *filters.SpeedtestUploadSpeedLt)
	}
	if filters.SpeedtestPingGt != nil {
		condition += fmt.Sprintf(" AND t.speedtest_ping >= %f", *filters.SpeedtestPingGt)
	}
	if filters.SpeedtestPingLt != nil {
		condition += fmt.Sprintf(" AND t.speedtest_ping <= %f", *filters.SpeedtestPingLt)
	}
	if filters.Country != nil && len(*filters.Country) >= 0 {
		resArgs = append(resArgs, "%"+*filters.Country+"%")
		condition += fmt.Sprintf(" AND t.country ILIKE $%d", len(resArgs))
	}
	if filters.ISP != nil && len(*filters.ISP) >= 0 {
		resArgs = append(resArgs, "%"+*filters.ISP+"%")
		condition += fmt.Sprintf(" AND t.isp ILIKE $%d", len(resArgs))
	}

	return
}
