package models

type Telemetry struct {
	StorageGitHash          string  `json:"storageGitHash"`
	ProviderGitHash         string  `json:"providerGitHash"`
	CPUName                 string  `json:"cpuName"`
	Country                 string  `json:"country"`
	ISP                     string  `json:"isp"`
	TotalProviderSpace      uint64  `json:"totalProviderSpace"`
	FreeProviderSpace       uint64  `json:"freeProviderSpace"`
	TotalRAM                uint64  `json:"totalRam"`
	FreeRAM                 uint64  `json:"freeRam"`
	BenchmarkDiskReadSpeed  float32 `json:"benchmarkDiskReadSpeed"`
	BenchmarkDiskWriteSpeed float32 `json:"benchmarkDiskWriteSpeed"`
	SpeedtestDownloadSpeed  float32 `json:"speedtestDownloadSpeed"`
	SpeedtestUploadSpeed    float32 `json:"speedtestUploadSpeed"`
	SpeedtestPing           float32 `json:"speedtestPing"`
	BenchmarkRocksOps       int32   `json:"benchmarkRocksOps"`
	CPUNumber               uint16  `json:"cpuNumber"`
	CPUIsVirtual            bool    `json:"cpuIsVirtual"`
}

type Provider struct {
	PubKey          string  `json:"pubkey"`
	MaxBagSizeBytes uint64  `json:"maxBagSizeBytes"`
	Rating          float32 `json:"rating"`      // rating from 0 to 5
	UpTime          float32 `json:"upTime"`      // percentage of uptime
	RegTime         uint32  `json:"regTime"`     // days since registration
	WorkingTime     uint32  `json:"workingTime"` // time since last offline in seconds
	Price           uint32  `json:"price"`       // price for storing 200GB per month
	MinSpan         uint32  `json:"minSpan"`     // min check interval in seconds
	MaxSpan         uint32  `json:"maxSpan"`     // max check interval in seconds
	IsSendTelemetry bool    `json:"isSendTelemetry"`

	Telemetry Telemetry `json:"telemetry"`
}

type ProvidersResponse struct {
	Providers []Provider `json:"providers"`
}
