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
	StorageGitHash               *string  `json:"storage_git_hash,omitempty"`
	ProviderGitHash              *string  `json:"provider_git_hash,omitempty"`
	CPUName                      *string  `json:"cpu_name,omitempty"`
	Location                     *string  `json:"location,omitempty"`
	Country                      *string  `json:"country,omitempty"`
	ISP                          *string  `json:"isp,omitempty"`
	RegTimeDaysGt                *int64   `json:"reg_time_days_gt,omitempty"`
	RegTimeDaysLt                *int64   `json:"reg_time_days_lt,omitempty"`
	WorkingTimeGtSec             *int64   `json:"working_time_gt_sec,omitempty"`
	WorkingTimeLtSec             *int64   `json:"working_time_lt_sec,omitempty"`
	MinSpanGt                    *int64   `json:"min_span_gt,omitempty"`
	MinSpanLt                    *int64   `json:"min_span_lt,omitempty"`
	MaxSpanGt                    *int64   `json:"max_span_gt,omitempty"`
	MaxSpanLt                    *int64   `json:"max_span_lt,omitempty"`
	MaxBagSizeBytesGt            *int64   `json:"max_bag_size_bytes_gt,omitempty"`
	MaxBagSizeBytesLt            *int64   `json:"max_bag_size_bytes_lt,omitempty"`
	BenchmarkDiskReadSpeedKiBGt  *int64   `json:"benchmark_disk_read_speed_gt,omitempty"`
	BenchmarkDiskReadSpeedKiBLt  *int64   `json:"benchmark_disk_read_speed_lt,omitempty"`
	BenchmarkDiskWriteSpeedKiBGt *int64   `json:"benchmark_disk_write_speed_gt,omitempty"`
	BenchmarkDiskWriteSpeedKiBLt *int64   `json:"benchmark_disk_write_speed_lt,omitempty"`
	SpeedtestDownloadSpeedGt     *int64   `json:"speedtest_download_gt,omitempty"`
	SpeedtestDownloadSpeedLt     *int64   `json:"speedtest_download_lt,omitempty"`
	SpeedtestUploadSpeedGt       *int64   `json:"speedtest_upload_gt,omitempty"`
	SpeedtestUploadSpeedLt       *int64   `json:"speedtest_upload_lt,omitempty"`
	SpeedtestPingGt              *int64   `json:"speedtest_ping_gt,omitempty"`
	SpeedtestPingLt              *int64   `json:"speedtest_ping_lt,omitempty"`
	RatingGt                     *float64 `json:"rating_gt,omitempty"`
	RatingLt                     *float64 `json:"rating_lt,omitempty"`
	UpTimeGtPercent              *float64 `json:"uptime_gt_percent,omitempty"`
	UpTimeLtPercent              *float64 `json:"uptime_lt_percent,omitempty"`
	PriceGt                      *float64 `json:"price_gt,omitempty"`
	PriceLt                      *float64 `json:"price_lt,omitempty"`
	TotalProviderSpaceGt         *float64 `json:"total_provider_space_gt,omitempty"`
	TotalProviderSpaceLt         *float64 `json:"total_provider_space_lt,omitempty"`
	UsedProviderSpaceGt          *float64 `json:"used_provider_space_gt,omitempty"`
	UsedProviderSpaceLt          *float64 `json:"used_provider_space_lt,omitempty"`
	CPUNumberGt                  *int32   `json:"cpu_number_gt,omitempty"`
	CPUNumberLt                  *int32   `json:"cpu_number_lt,omitempty"`
	TotalRamGt                   *float32 `json:"total_ram_gt,omitempty"`
	TotalRamLt                   *float32 `json:"total_ram_lt,omitempty"`
	UsageRamPercentGt            *float32 `json:"ram_usage_percent_gt,omitempty"`
	UsageRamPercentLt            *float32 `json:"ram_usage_percent_lt,omitempty"`
	CPUIsVirtual                 *bool    `json:"cpu_is_virtual,omitempty"`
	IsSendTelemetry              *bool    `json:"is_send_telemetry,omitempty"`
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
	PublicKey          string      `json:"public_key" db:"public_key"`
	StorageGitHash     string      `json:"storage_git_hash" db:"storage_git_hash"`
	ProviderGitHash    string      `json:"provider_git_hash" db:"provider_git_hash"`
	CPUName            string      `json:"cpu_name" db:"cpu_name"`
	Pings              string      `json:"pings" db:"pings"`
	CPUProductName     string      `json:"cpu_product_name" db:"cpu_product_name"`
	USysname           string      `json:"uname_sysname" db:"uname_sysname"`
	URelease           string      `json:"uname_release" db:"uname_release"`
	UVersion           string      `json:"uname_version" db:"uname_version"`
	UMachine           string      `json:"uname_machine" db:"uname_machine"`
	DiskName           string      `json:"disk_name" db:"disk_name"`
	CPULoad            []float32   `json:"cpu_load" db:"cpu_load"`
	TotalDiskSpace     float64     `json:"total_space" db:"total_space"`
	FreeDiskSpace      float64     `json:"free_space" db:"free_space"`
	UsedDiskSpace      float64     `json:"used_space" db:"used_space"`
	UsedProviderSpace  float64     `json:"used_provider_space" db:"used_provider_space"`
	TotalProviderSpace float64     `json:"total_provider_space" db:"total_provider_space"`
	SwapTotal          float32     `json:"total_swap" db:"total_swap"`
	SwapUsage          float32     `json:"usage_swap" db:"usage_swap"`
	SwapUsagePercent   float32     `json:"swap_usage_percent" db:"swap_usage_percent"`
	RAMUsage           float32     `json:"usage_ram" db:"usage_ram"`
	RAMTotal           float32     `json:"total_ram" db:"total_ram"`
	RAMUsagePercent    float32     `json:"ram_usage_percent" db:"ram_usage_percent"`
	MaxBagSizeBytes    uint64      `json:"max_bag_size_bytes" db:"max_bag_size_bytes"`
	CPUNumber          int32       `json:"cpu_number" db:"cpu_number"`
	CPUIsVirtual       bool        `json:"cpu_is_virtual" db:"cpu_is_virtual"`
	TelemetryIP        string      `json:"x_real_ip" db:"x_real_ip"`
	NetLoad            []float32   `json:"net_load"`
	NetReceived        []float32   `json:"net_recv"`
	NetSent            []float32   `json:"net_sent"`
	DisksLoad          interface{} `json:"disks_load"`
	DisksLoadPercent   interface{} `json:"disks_load_percent"`
	IOPS               interface{} `json:"iops"`
	PPS                []float32   `json:"pps"`
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

type FiltersRange struct {
	Locations                  []string
	RegTimeDaysMax             int64
	MinSpanMin                 int64
	MinSpanMax                 int64
	MaxSpanMin                 int64
	MaxSpanMax                 int64
	MaxBagSizeMbMin            int64
	MaxBagSizeMbMax            int64
	BenchmarkDiskReadSpeedMin  int64
	BenchmarkDiskReadSpeedMax  int64
	BenchmarkDiskWriteSpeedMin int64
	BenchmarkDiskWriteSpeedMax int64
	SpeedtestDownloadSpeedMin  int64
	SpeedtestDownloadSpeedMax  int64
	SpeedtestUploadSpeedMin    int64
	SpeedtestUploadSpeedMax    int64
	TotalProviderSpaceMin      int64
	TotalProviderSpaceMax      int64
	UsedProviderSpaceMax       int64
	RatingMax                  float64
	PriceMax                   float64
	CPUNumberMax               int32
	SpeedtestPingMin           int32
	SpeedtestPingMax           int32
	TotalRAMMin                float32
	TotalRAMMax                float32
}

type ProviderDB struct {
	Location    *Location `json:"location"`
	Status      *uint32   `json:"status"`
	PubKey      string    `json:"public_key"`
	Address     string    `json:"address"`
	UpTime      float32   `json:"uptime"`
	Rating      float32   `json:"rating"`
	StatusRatio float32   `json:"status_ratio"`
	MaxSpan     uint32    `json:"max_span"`
	Price       uint64    `json:"price"`

	MinSpan         uint32      `json:"min_span"`
	MaxBagSizeBytes uint64      `json:"max_bag_size_bytes"`
	RegTime         uint64      `json:"registered_at"`
	IsSendTelemetry bool        `json:"is_send_telemetry"`
	Telemetry       TelemetryDB `json:"telemetry"`
}

type Location struct {
	Country    string `json:"country"`
	CountryISO string `json:"country_iso"`
	City       string `json:"city"`
	TimeZone   string `json:"time_zone"`
}

type ProviderWallet struct {
	PubKey  string `db:"public_key"`
	Address string `db:"address"`
	LT      uint64 `db:"last_tx_lt"`
}

type ProviderWalletLT struct {
	PubKey string `db:"public_key"`
	LT     uint64 `db:"last_tx_lt"`
}

type ContractToProviderRelation struct {
	ProviderPublicKey string `json:"provider_public_key"`
	ProviderAddress   string `json:"provider_address"`
	Address           string `json:"address"`
	Size              uint64 `json:"size"`
}

type StorageContract struct {
	ProvidersAddresses map[string]struct{} `json:"providers_addresses"`
	Address            string              `json:"address"`
	BagID              string              `json:"bag_id"`
	OwnerAddr          string              `json:"owner_address"`
	Size               uint64              `json:"size"`
	ChunkSize          uint64              `json:"chunk_size"`
	LastLT             uint64              `json:"last_tx_lt"`
}

type ProviderIP struct {
	PublicKey string `json:"public_key"`
	Storage   IPInfo `json:"storage"`
	Provider  IPInfo `json:"provider"`
}

type IPInfo struct {
	IP   string `json:"ip"`
	Port int32  `json:"port"`
}

type ProviderIPInfo struct {
	PublicKey string `json:"public_key"`
	IPInfo    string `json:"ip_info"`
}

type ContractProofsCheck struct {
	Address         string    `json:"address"`
	ProviderAddress string    `json:"provider_address"`
	Reason          uint32    `json:"reason"`
	Timestamp       time.Time `json:"timestamp"`
}

type ContractCheck struct {
	Address           string
	ProviderPublicKey string
	ReasonTimestamp   *time.Time
	Reason            *uint32
}
