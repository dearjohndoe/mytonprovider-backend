package providers

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"mytonprovider-backend/pkg/models/db"
)

const (
	providersQuerySelect = `
		SELECT 
			p.public_key,
			p.address,
			p.status,
			p.uptime * 100 as uptime,
			p.rating,
			p.max_span,
			p.rate_per_mb_per_day * 1024 * 200 * 30 as price,
			p.min_span,
			p.max_bag_size_bytes,
			p.registered_at,
			t.public_key is not null as is_send_telemetry,
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
			t.updated_at,
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
		WHERE p.is_initialized AND p.rating IS NOT NULL AND p.uptime IS NOT NULL
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
		// Convert price from TON to rate_per_mb_per_day
		condition += fmt.Sprintf(" AND p.rate_per_mb_per_day >= %f", *filters.PriceGt*1000000000/1024/200/30)
	}
	if filters.PriceLt != nil {
		// Convert price from TON to rate_per_mb_per_day
		condition += fmt.Sprintf(" AND p.rate_per_mb_per_day <= %f", *filters.PriceLt*1000000000/1024/200/30)
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
	if filters.MaxBagSizeBytesGt != nil {
		condition += fmt.Sprintf(" AND p.max_bag_size_bytes/1024/1024 >= %d", *filters.MaxBagSizeBytesGt)
	}
	if filters.MaxBagSizeBytesLt != nil {
		condition += fmt.Sprintf(" AND p.max_bag_size_bytes/1024/1024 <= %d + 1", *filters.MaxBagSizeBytesLt)
	}
	if filters.IsSendTelemetry != nil {
		if *filters.IsSendTelemetry {
			condition += " AND t.public_key is not null"
		} else {
			condition += " AND t.public_key is null"
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
	if filters.BenchmarkDiskReadSpeedKiBGt != nil {
		condition += fmt.Sprintf(" AND providers.parse_speed_to_int(b.qd64_disk_read_speed) >= %d", *filters.BenchmarkDiskReadSpeedKiBGt*1024)
	}
	if filters.BenchmarkDiskReadSpeedKiBLt != nil {
		condition += fmt.Sprintf(" AND providers.parse_speed_to_int(b.qd64_disk_read_speed) <= %d", *filters.BenchmarkDiskReadSpeedKiBLt*1024)
	}
	if filters.BenchmarkDiskWriteSpeedKiBGt != nil {
		condition += fmt.Sprintf(" AND providers.parse_speed_to_int(b.qd64_disk_write_speed) >= %d", *filters.BenchmarkDiskWriteSpeedKiBGt*1024)
	}
	if filters.BenchmarkDiskWriteSpeedKiBLt != nil {
		condition += fmt.Sprintf(" AND providers.parse_speed_to_int(b.qd64_disk_write_speed) <= %d", *filters.BenchmarkDiskWriteSpeedKiBLt*1024)
	}
	if filters.SpeedtestDownloadSpeedGt != nil {
		condition += fmt.Sprintf(" AND b.speedtest_download >= %d", *filters.SpeedtestDownloadSpeedGt)
	}
	if filters.SpeedtestDownloadSpeedLt != nil {
		condition += fmt.Sprintf(" AND b.speedtest_download <= %d", *filters.SpeedtestDownloadSpeedLt)
	}
	if filters.SpeedtestUploadSpeedGt != nil {
		condition += fmt.Sprintf(" AND b.speedtest_upload >= %d", *filters.SpeedtestUploadSpeedGt)
	}
	if filters.SpeedtestUploadSpeedLt != nil {
		condition += fmt.Sprintf(" AND b.speedtest_upload <= %d", *filters.SpeedtestUploadSpeedLt)
	}
	if filters.SpeedtestPingGt != nil {
		condition += fmt.Sprintf(" AND b.speedtest_ping >= %d", *filters.SpeedtestPingGt)
	}
	if filters.SpeedtestPingLt != nil {
		condition += fmt.Sprintf(" AND b.speedtest_ping <= %d", *filters.SpeedtestPingLt)
	}
	if filters.Country != nil && len(*filters.Country) >= 0 {
		resArgs = append(resArgs, "%"+*filters.Country+"%")
		condition += fmt.Sprintf(" AND b.country ILIKE $%d", len(resArgs))
	}
	if filters.ISP != nil && len(*filters.ISP) >= 0 {
		resArgs = append(resArgs, "%"+*filters.ISP+"%")
		condition += fmt.Sprintf(" AND b.isp ILIKE $%d", len(resArgs))
	}

	return
}

func scanProviderDBRows(rows pgx.Rows) (providers []db.ProviderDB, err error) {
	for rows.Next() {
		var regTime time.Time
		var updatedAt *time.Time
		var provider db.ProviderDB
		if err := rows.Scan(
			&provider.PubKey,
			&provider.Address,
			&provider.Status,
			&provider.UpTime,
			&provider.Rating,
			&provider.MaxSpan,
			&provider.Price,
			&provider.MinSpan,
			&provider.MaxBagSizeBytes,
			&regTime,
			&provider.IsSendTelemetry,
			&provider.Telemetry.StorageGitHash,
			&provider.Telemetry.ProviderGitHash,
			&provider.Telemetry.TotalProviderSpace,
			&provider.Telemetry.UsedProviderSpace,
			&provider.Telemetry.CPUName,
			&provider.Telemetry.CPUNumber,
			&provider.Telemetry.CPUIsVirtual,
			&provider.Telemetry.TotalRAM,
			&provider.Telemetry.UsageRAM,
			&provider.Telemetry.UsageRAMPercent,
			&updatedAt,
			&provider.Telemetry.BenchmarkDiskReadSpeed,
			&provider.Telemetry.BenchmarkDiskWriteSpeed,
			&provider.Telemetry.SpeedtestDownload,
			&provider.Telemetry.SpeedtestUpload,
			&provider.Telemetry.SpeedtestPing,
			&provider.Telemetry.Country,
			&provider.Telemetry.ISP); err != nil {
			return nil, err
		}

		if updatedAt != nil {
			u := uint64(updatedAt.Unix())
			provider.Telemetry.UpdatedAt = &u
		}

		provider.RegTime = uint64(regTime.Unix())
		providers = append(providers, provider)
	}

	err = rows.Err()

	return
}
