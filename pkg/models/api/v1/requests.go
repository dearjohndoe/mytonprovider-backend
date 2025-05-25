package v1

type TelemetryRequest struct {
	Storage          StorageInfo        `json:"storage"`
	GitHashes        map[string]string  `json:"git_hashes"`
	NetLoad          interface{}        `json:"net_load"`           // todo: define
	DisksLoad        interface{}        `json:"disks_load"`         // todo: define
	DisksLoadPercent interface{}        `json:"disks_load_percent"` // todo: define
	IOPS             interface{}        `json:"iops"`               // todo: define
	PPS              interface{}        `json:"pps"`                // todo: define
	Memory           MemoryInfo         `json:"memory"`
	Swap             MemoryInfo         `json:"swap"`
	Uname            UnameInfo          `json:"uname"`
	CPUInfo          CPUInfo            `json:"cpu_info"`
	Pings            map[string]float64 `json:"pings"`
	Benchmark        BenchmarkInfo      `json:"benchmark"`
}

type ProviderInfo struct {
	PubKey             string  `json:"pubkey"`
	UsedProviderSpace  float64 `json:"used_provider_space"`
	TotalProviderSpace float64 `json:"total_provider_space"`
}

type StorageInfo struct {
	PubKey         string       `json:"pubkey"`
	DiskName       string       `json:"disk_name"`
	TotalDiskSpace int64        `json:"total_disk_space"`
	UsedDiskSpace  int64        `json:"used_disk_space"`
	FreeDiskSpace  int64        `json:"free_disk_space"`
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
	Number      int32     `json:"number"`
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
