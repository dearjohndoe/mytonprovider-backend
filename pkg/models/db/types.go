package db

import "time"

type ProviderUpdate struct {
	Pubkey       string `json:"public_key"`
	RatePerMBDay int64  `json:"rate_per_mb_per_day"`
	MinBounty    int64  `json:"min_bounty"`
	MinSpan      uint32 `json:"min_span"`
	MaxSpan      uint32 `json:"max_span"`
}

type ProviderCreate struct {
	Pubkey       string    `json:"public_key"`
	Address      string    `json:"address"`
	RegisteredAt time.Time `json:"registered_at"`
}

type ProviderFilters struct {
	RatingGt                  *float64 `json:"rating_gt,omitempty"`
	RatingLt                  *float64 `json:"rating_lt,omitempty"`
	RegTimeDaysGt             *int64   `json:"reg_time_days_gt,omitempty"`
	RegTimeDaysLt             *int64   `json:"reg_time_days_lt,omitempty"`
	UpTimeGtPercent           *float64 `json:"uptime_gt_percent,omitempty"`
	UpTimeLtPercent           *float64 `json:"uptime_lt_percent,omitempty"`
	WorkingTimeGtSec          *int64   `json:"working_time_gt_sec,omitempty"`
	WorkingTimeLtSec          *int64   `json:"working_time_lt_sec,omitempty"`
	PriceGt                   *float64 `json:"price_gt,omitempty"`
	PriceLt                   *float64 `json:"price_lt,omitempty"`
	MinSpanGt                 *int64   `json:"min_span_gt,omitempty"`
	MinSpanLt                 *int64   `json:"min_span_lt,omitempty"`
	MaxSpanGt                 *int64   `json:"max_span_gt,omitempty"`
	MaxSpanLt                 *int64   `json:"max_span_lt,omitempty"`
	MaxBagSizeBytesGt         *int64   `json:"max_bag_size_bytes_gt,omitempty"`
	MaxBagSizeBytesLt         *int64   `json:"max_bag_size_bytes_lt,omitempty"`
	IsSendTelemetry           *bool    `json:"is_send_telemetry,omitempty"`
	TotalProviderSpaceGt      *float64 `json:"total_provider_space_gt,omitempty"`
	TotalProviderSpaceLt      *float64 `json:"total_provider_space_lt,omitempty"`
	UsedProviderSpaceGt       *float64 `json:"used_provider_space_gt,omitempty"`
	UsedProviderSpaceLt       *float64 `json:"used_provider_space_lt,omitempty"`
	StorageGitHash            *string  `json:"storage_git_hash,omitempty"`
	ProviderGitHash           *string  `json:"provider_git_hash,omitempty"`
	CPUNumberGt               *int32   `json:"cpu_number_gt,omitempty"`
	CPUNumberLt               *int32   `json:"cpu_number_lt,omitempty"`
	CPUName                   *string  `json:"cpu_name,omitempty"`
	CPUIsVirtual              *bool    `json:"cpu_is_virtual,omitempty"`
	TotalRamGt                *float32 `json:"total_ram_gt,omitempty"`
	TotalRamLt                *float32 `json:"total_ram_lt,omitempty"`
	UsageRamPercentGt         *float32 `json:"ram_usage_percent_gt,omitempty"`
	UsageRamPercentLt         *float32 `json:"ram_usage_percent_lt,omitempty"`
	BenchmarkDiskReadSpeedGt  *float64 `json:"benchmark_disk_read_speed_gt,omitempty"`
	BenchmarkDiskReadSpeedLt  *float64 `json:"benchmark_disk_read_speed_lt,omitempty"`
	BenchmarkDiskWriteSpeedGt *float64 `json:"benchmark_disk_write_speed_gt,omitempty"`
	BenchmarkDiskWriteSpeedLt *float64 `json:"benchmark_disk_write_speed_lt,omitempty"`
	SpeedtestDownloadSpeedGt  *float64 `json:"speedtest_download_gt,omitempty"`
	SpeedtestDownloadSpeedLt  *float64 `json:"speedtest_download_lt,omitempty"`
	SpeedtestUploadSpeedGt    *float64 `json:"speedtest_upload_gt,omitempty"`
	SpeedtestUploadSpeedLt    *float64 `json:"speedtest_upload_lt,omitempty"`
	SpeedtestPingGt           *float64 `json:"speedtest_ping_gt,omitempty"`
	SpeedtestPingLt           *float64 `json:"speedtest_ping_lt,omitempty"`
	Country                   *string  `json:"country,omitempty"`
	ISP                       *string  `json:"isp,omitempty"`
}

type ProviderSort struct {
	Column string `json:"column,omitempty"` // "rating", "price", "uptime", "maxSpan" or "workingtime"
	Order  string `json:"order,omitempty"`  // "asc" or "desc"
}

type ProviderStatusUpdate struct {
	Pubkey   string `json:"public_key"`
	IsOnline bool   `json:"is_online"`
}

type BenchmarkUpdate struct {
	PublicKey          string  `json:"public_key" db:"public_key"`
	Disk               string  `json:"disk" db:"disk"`
	Network            string  `json:"network" db:"network"`
	DiskReadSpeed      string  `json:"qd64_disk_read_speed" db:"qd64_disk_read_speed"`   // MiB/s
	DiskWriteSpeed     string  `json:"qd64_disk_write_speed" db:"qd64_disk_write_speed"` // MiB/s
	BenchmarkTimestamp string  `json:"benchmark_timestamp" db:"benchmark_timestamp"`     // RFC3339
	SpeedtestDownload  float64 `json:"speedtest_download" db:"speedtest_download"`
	SpeedtestUpload    float64 `json:"speedtest_upload" db:"speedtest_upload"`
	SpeedtestPing      float32 `json:"speedtest_ping" db:"speedtest_ping"` // ms
	Country            string  `json:"country" db:"country"`
	ISP                string  `json:"isp" db:"isp"`
}

type TelemetryUpdate struct {
	PublicKey          string    `json:"public_key" db:"public_key"`
	StorageGitHash     string    `json:"storage_git_hash" db:"storage_git_hash"`
	ProviderGitHash    string    `json:"provider_git_hash" db:"provider_git_hash"`
	CPUName            string    `json:"cpu_name" db:"cpu_name"`
	Pings              string    `json:"pings" db:"pings"`
	CPUProductName     string    `json:"cpu_product_name" db:"cpu_product_name"`
	USysname           string    `json:"uname_sysname" db:"uname_sysname"`
	URelease           string    `json:"uname_release" db:"uname_release"`
	UVersion           string    `json:"uname_version" db:"uname_version"`
	UMachine           string    `json:"uname_machine" db:"uname_machine"`
	DiskName           string    `json:"disk_name" db:"disk_name"`
	CPULoad            []float32 `json:"cpu_load" db:"cpu_load"`
	TotalDiskSpace     float64   `json:"total_space" db:"total_space"`
	FreeDiskSpace      float64   `json:"free_space" db:"free_space"`
	UsedDiskSpace      float64   `json:"used_space" db:"used_space"`
	UsedProviderSpace  float64   `json:"used_provider_space" db:"used_provider_space"`
	TotalProviderSpace float64   `json:"total_provider_space" db:"total_provider_space"`
	SwapTotal          float32   `json:"total_swap" db:"total_swap"`
	SwapUsage          float32   `json:"usage_swap" db:"usage_swap"`
	SwapUsagePercent   float32   `json:"swap_usage_percent" db:"swap_usage_percent"`
	RAMUsage           float32   `json:"usage_ram" db:"usage_ram"`
	RAMTotal           float32   `json:"total_ram" db:"total_ram"`
	RAMUsagePercent    float32   `json:"ram_usage_percent" db:"ram_usage_percent"`
	MaxBagSizeBytes    uint64    `json:"max_bag_size_bytes" db:"max_bag_size_bytes"`
	CPUNumber          int32     `json:"cpu_number" db:"cpu_number"`
	CPUIsVirtual       bool      `json:"cpu_is_virtual" db:"cpu_is_virtual"`
}

type TelemetryDB struct {
	StorageGitHash          *string  `json:"storage_git_hash"`
	ProviderGitHash         *string  `json:"provider_git_hash"`
	BenchmarkDiskReadSpeed  *string  `json:"qd64_disk_read_speed"`
	BenchmarkDiskWriteSpeed *string  `json:"qd64_disk_write_speed"`
	CPUName                 *string  `json:"cpu_name"`
	Country                 *string  `json:"country"`
	ISP                     *string  `json:"isp"`
	UpdatedAt               *uint64  `json:"updated_at"`
	TotalProviderSpace      *float32 `json:"total_provider_space"`
	UsedProviderSpace       *float32 `json:"used_provider_space"`
	TotalRAM                *float32 `json:"total_ram"`
	UsageRAM                *float32 `json:"usage_ram"`
	UsageRAMPercent         *float32 `json:"ram_usage_percent"`
	SpeedtestDownload       *float32 `json:"speedtest_download"`
	SpeedtestUpload         *float32 `json:"speedtest_upload"`
	SpeedtestPing           *float32 `json:"speedtest_ping"`
	CPUNumber               *uint16  `json:"cpu_number"`
	CPUIsVirtual            *bool    `json:"cpu_is_virtual"`
}

type ProviderDB struct {
	PubKey  string  `json:"public_key"`
	UpTime  float32 `json:"uptime"`
	Rating  float32 `json:"rating"`
	MaxSpan uint32  `json:"max_span"`
	Price   uint64  `json:"price"`

	MinSpan         uint32      `json:"min_span"`
	MaxBagSizeBytes uint64      `json:"max_bag_size_bytes"`
	RegTime         uint64      `json:"registered_at"`
	IsSendTelemetry bool        `json:"is_send_telemetry"`
	Telemetry       TelemetryDB `json:"telemetry"`
}

type ProviderWallet struct {
	PubKey  string `db:"public_key"`
	Address string `db:"address"`
	LT      uint64 `db:"lt"`
}
