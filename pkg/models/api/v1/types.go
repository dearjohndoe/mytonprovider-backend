package v1

type SearchProvidersRequest struct {
	Filters Filters `json:"filters,omitempty"`
	Sort    Sort    `json:"sort,omitempty"`
	Limit   int     `json:"limit,omitempty"`
	Offset  int     `json:"offset,omitempty"`

	// If set, only providers with exact match the pubkey column will be returned.
	Exact []string `json:"exact,omitempty"`
}

type Sort struct {
	Column string `json:"column,omitempty"` // "workingtime", "rating", "price", "uptime" or "maxSpan"
	Order  string `json:"order,omitempty"`  // "asc" or "desc"
}

type Filters struct {
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
	UsageRamPercentGt         *float32 `json:"usage_ram_percent_gt,omitempty"`
	UsageRamPercentLt         *float32 `json:"usage_ram_percent_lt,omitempty"`
	BenchmarkDiskReadSpeedGt  *float64 `json:"benchmark_disk_read_speed_gt,omitempty"`
	BenchmarkDiskReadSpeedLt  *float64 `json:"benchmark_disk_read_speed_lt,omitempty"`
	BenchmarkDiskWriteSpeedGt *float64 `json:"benchmark_disk_write_speed_gt,omitempty"`
	BenchmarkDiskWriteSpeedLt *float64 `json:"benchmark_disk_write_speed_lt,omitempty"`
	SpeedtestDownloadSpeedGt  *float64 `json:"speedtest_download_speed_gt,omitempty"`
	SpeedtestDownloadSpeedLt  *float64 `json:"speedtest_download_speed_lt,omitempty"`
	SpeedtestUploadSpeedGt    *float64 `json:"speedtest_upload_speed_gt,omitempty"`
	SpeedtestUploadSpeedLt    *float64 `json:"speedtest_upload_speed_lt,omitempty"`
	SpeedtestPingGt           *float64 `json:"speedtest_ping_gt,omitempty"`
	SpeedtestPingLt           *float64 `json:"speedtest_ping_lt,omitempty"`
	Country                   *string  `json:"country,omitempty"`
	ISP                       *string  `json:"isp,omitempty"`
}

type TelemetryRequest struct {
	Storage          StorageInfo        `json:"storage"`
	GitHashes        map[string]string  `json:"git_hashes"`
	NetLoad          interface{}        `json:"net_load"`           // todo: define
	DisksLoad        interface{}        `json:"disks_load"`         // todo: define
	DisksLoadPercent interface{}        `json:"disks_load_percent"` // todo: define
	IOPS             interface{}        `json:"iops"`               // todo: define
	PPS              interface{}        `json:"pps"`                // todo: define
	Memory           MemoryInfo         `json:"ram"`
	Swap             MemoryInfo         `json:"swap"`
	Uname            UnameInfo          `json:"uname"`
	CPUInfo          CPUInfo            `json:"cpu_info"`
	Pings            map[string]float32 `json:"pings"`
	Benchmark        interface{}        `json:"benchmark"`
}

type DiskBenchmark struct {
	Name      string `json:"name"`
	Read      string `json:"read"`
	Write     string `json:"write"`
	ReadIOPS  string `json:"read_iops"`
	WriteIOPS string `json:"write_iops"`
}

type ClientInfo struct {
	Country        string `json:"country"`
	IP             string `json:"ip"`
	ISP            string `json:"isp"`
	ISPDownloadAvg string `json:"ispdlavg"`
	ISPUploadAvg   string `json:"ispulavg"`
	ISPRating      string `json:"isprating"`
	Lat            string `json:"lat"`
	Lon            string `json:"lon"`
	LoggedIn       string `json:"loggedin"`
	Rating         string `json:"rating"`
}

type ServerInfo struct {
	CC      string  `json:"cc"`
	Country string  `json:"country"`
	D       float64 `json:"d"`
	Host    string  `json:"host"`
	ID      string  `json:"id"`
	Lat     string  `json:"lat"`
	Lon     string  `json:"lon"`
	Name    string  `json:"name"`
	Sponsor string  `json:"sponsor"`
	URL     string  `json:"url"`
	Latency float32 `json:"latency"`
}

type NetworkBenchmark struct {
	Server        ServerInfo `json:"server"`
	Client        ClientInfo `json:"client"`
	Share         string     `json:"share"`
	Timestamp     string     `json:"timestamp"` // RFC3339
	BytesReceived uint64     `json:"bytes_received"`
	BytesSent     uint64     `json:"bytes_sent"`
	Download      float64    `json:"download"`
	Upload        float64    `json:"upload"`
	Ping          float32    `json:"ping"`
}

type BenchmarksRequest struct {
	PubKey    string                   `json:"pubkey"`
	Disk      map[string]DiskBenchmark `json:"disk"`
	Network   NetworkBenchmark         `json:"network"`
	Timestamp int64                    `json:"timestamp"` // Unix timestamp in seconds
}

type ProviderInfo struct {
	PubKey             string  `json:"pubkey"`
	UsedProviderSpace  float64 `json:"used_provider_space"`
	TotalProviderSpace float64 `json:"total_provider_space"`
}

type StorageInfo struct {
	PubKey         string       `json:"pubkey"`
	DiskName       string       `json:"disk_name"`
	TotalDiskSpace float64      `json:"total_disk_space"`
	UsedDiskSpace  float64      `json:"used_disk_space"`
	FreeDiskSpace  float64      `json:"free_disk_space"`
	Provider       ProviderInfo `json:"provider"`
}

type MemoryInfo struct {
	Total        float32 `json:"total"`
	Usage        float32 `json:"usage"`
	UsagePercent float32 `json:"usage_percent"`
}

type UnameInfo struct {
	Sysname string `json:"sysname"`
	Release string `json:"release"`
	Version string `json:"version"`
	Machine string `json:"machine"`
}

type CPUInfo struct {
	CPULoad     []float32 `json:"cpu_load"`
	Number      int32     `json:"cpu_count"`
	CPUName     string    `json:"cpu_name"`
	ProductName string    `json:"product_name"`
	IsVirtual   bool      `json:"is_virtual"`
}

type BenchmarkLevel struct {
	ReadSpeed  *float64 `json:"read_speed"`
	WriteSpeed *float64 `json:"write_speed"`
	ReadIOPS   *float64 `json:"read_iops"`
	WriteIOPS  *float64 `json:"write_iops"`
	RandomOps  *float64 `json:"random_ops"`
}

type BenchmarkInfo struct {
	Lite BenchmarkLevel `json:"lite"`
	Hard BenchmarkLevel `json:"hard"`
	Full BenchmarkLevel `json:"full"`
}

type Telemetry struct {
	StorageGitHash          *string  `json:"storage_git_hash"`
	ProviderGitHash         *string  `json:"provider_git_hash"`
	BenchmarkDiskReadSpeed  *string  `json:"qd64_disk_read_speed"`
	BenchmarkDiskWriteSpeed *string  `json:"qd64_disk_write_speed"`
	Country                 *string  `json:"country"`
	ISP                     *string  `json:"isp"`
	CPUName                 *string  `json:"cpu_name"`
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

type Provider struct {
	PubKey      string  `json:"pubkey"`
	UpTime      float32 `json:"uptime"`
	WorkingTime uint64  `json:"working_time"`
	Rating      float32 `json:"rating"`
	MaxSpan     uint32  `json:"max_span"`
	Price       uint64  `json:"price"`

	MinSpan         uint32    `json:"min_span"`
	MaxBagSizeBytes uint64    `json:"max_bag_size_bytes"`
	RegTime         uint64    `json:"reg_time"`
	IsSendTelemetry bool      `json:"is_send_telemetry"`
	Telemetry       Telemetry `json:"telemetry"`
}

type ProvidersResponse struct {
	Providers []Provider `json:"providers"`
}

type TelemetryResponse struct {
	PubKey    string    `json:"pubkey"`
	Telemetry Telemetry `json:"telemetry"`
}
