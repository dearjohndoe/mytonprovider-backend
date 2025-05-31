package db

import "time"

type Provider struct {
	PubKey string `json:"storage_pub_key" db:"storage_pub_key"`
	Telemetry
}

type ProviderInfo struct {
	Pubkey       string  `json:"public_key"`
	Rating       float64 `json:"rating"`
	RatePerMBDay int64   `json:"rate_per_mb_per_day"`
	MinBounty    int64   `json:"min_bounty"`
	MinSpan      uint32  `json:"min_span"`
	MaxSpan      uint32  `json:"max_span"`
}

type ProviderInit struct {
	Pubkey       string    `json:"public_key"`
	Address      string    `json:"address"`
	RegisteredAt time.Time `json:"registered_at"`
}

type ProviderFilters struct{}

type ProviderSort struct{}

type Telemetry struct {
	PublicKey               string    `json:"public_key" db:"public_key"`
	StorageGitHash          string    `json:"storage_git_hash" db:"storage_git_hash"`
	ProviderGitHash         string    `json:"provider_git_hash" db:"provider_git_hash"`
	CPUName                 string    `json:"cpu_name" db:"cpu_name"`
	Country                 string    `json:"country" db:"country"`
	ISP                     string    `json:"isp" db:"isp"`
	Pings                   string    `json:"pings" db:"pings"`
	Benchmarks              string    `json:"benchmarks" db:"benchmarks"`
	CPUProductName          string    `json:"cpu_product_name" db:"cpu_product_name"`
	USysname                string    `json:"uname_sysname" db:"uname_sysname"`
	URelease                string    `json:"uname_release" db:"uname_release"`
	UVersion                string    `json:"uname_version" db:"uname_version"`
	UMachine                string    `json:"uname_machine" db:"uname_machine"`
	DiskName                string    `json:"disk_name" db:"disk_name"`
	CPULoad                 []float32 `json:"cpu_load" db:"cpu_load"`
	TotalDiskSpace          float64   `json:"total_space" db:"total_space"`
	FreeDiskSpace           float64   `json:"free_space" db:"free_space"`
	UsedDiskSpace           float64   `json:"used_space" db:"used_space"`
	BenchmarkDiskReadSpeed  int64     `json:"benchmark_disk_read_speed" db:"benchmark_disk_read_speed"`
	BenchmarkDiskWriteSpeed int64     `json:"benchmark_disk_write_speed" db:"benchmark_disk_write_speed"`
	BenchmarkRocksOps       int64     `json:"benchmark_rocks_ops" db:"benchmark_rocks_ops"`
	SpeedtestDownloadSpeed  float64   `json:"speedtest_download_speed" db:"speedtest_download_speed"`
	SpeedtestUploadSpeed    float64   `json:"speedtest_upload_speed" db:"speedtest_upload_speed"`
	SpeedtestPing           float64   `json:"speedtest_ping" db:"speedtest_ping"`
	UsedProviderSpace       float64   `json:"used_provider_space" db:"used_provider_space"`
	TotalProviderSpace      float64   `json:"total_provider_space" db:"total_provider_space"`
	SwapTotal               float32   `json:"total_swap" db:"total_swap"`
	SwapFree                float32   `json:"free_swap" db:"free_swap"`
	SwapUsagePercent        float32   `json:"swap_usage_percent" db:"swap_usage_percent"`
	RAMFree                 float32   `json:"free_ram" db:"free_ram"`
	RAMTotal                float32   `json:"total_ram" db:"total_ram"`
	RAMUsagePercent         float32   `json:"ram_usage_percent" db:"ram_usage_percent"`
	CPUNumber               int32     `json:"cpu_number" db:"cpu_number"`
	CPUIsVirtual            bool      `json:"cpu_is_virtual" db:"cpu_is_virtual"`
}
