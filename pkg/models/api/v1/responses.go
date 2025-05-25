package v1

type Telemetry struct {
	StorageGitHash          string  `json:"storage_git_hash"`
	ProviderGitHash         string  `json:"provider_git_hash"`
	CPUName                 string  `json:"cpu_name"`
	Country                 string  `json:"country"`
	ISP                     string  `json:"isp"`
	TotalProviderSpace      uint64  `json:"total_provider_space"`
	FreeProviderSpace       uint64  `json:"free_provider_space"`
	TotalRAM                uint64  `json:"total_ram"`
	FreeRAM                 uint64  `json:"free_ram"`
	BenchmarkDiskReadSpeed  float32 `json:"benchmark_disk_read_speed"`
	BenchmarkDiskWriteSpeed float32 `json:"benchmark_disk_write_speed"`
	SpeedtestDownloadSpeed  float32 `json:"speedtest_download_speed"`
	SpeedtestUploadSpeed    float32 `json:"speedtest_upload_speed"`
	SpeedtestPing           float32 `json:"speedtest_ping"`
	BenchmarkRocksOps       int32   `json:"benchmark_rocks_ops"`
	CPUNumber               uint16  `json:"cpu_number"`
	CPUIsVirtual            bool    `json:"cpu_is_virtual"`
}

type Provider struct {
	PubKey          string  `json:"pubkey"`
	MaxBagSizeBytes uint64  `json:"max_bag_size_bytes"`
	Rating          float32 `json:"rating"`
	UpTime          float32 `json:"up_time"`
	RegTime         uint32  `json:"reg_time"`
	WorkingTime     uint32  `json:"working_time"`
	Price           uint32  `json:"price"`
	MinSpan         uint32  `json:"min_span"`
	MaxSpan         uint32  `json:"max_span"`
	IsSendTelemetry bool    `json:"is_send_telemetry"`

	Telemetry Telemetry `json:"telemetry"`
}

type ProvidersResponse struct {
	Providers []Provider `json:"providers"`
}

type TelemetryResponse struct {
	PubKey    string    `json:"pubkey"`
	Telemetry Telemetry `json:"telemetry"`
}
