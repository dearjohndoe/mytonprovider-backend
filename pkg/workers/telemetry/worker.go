package telemetry

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"mytonprovider-backend/pkg/cache"
	v1 "mytonprovider-backend/pkg/models/api/v1"
	"mytonprovider-backend/pkg/models/db"
)

type providers interface {
	GetAllProvidersPubkeys(ctx context.Context) (pubkeys []string, err error)
	UpdateTelemetry(ctx context.Context, telemetry []db.TelemetryUpdate) (err error)
	UpdateBenchmarks(ctx context.Context, benchmarks []db.BenchmarkUpdate) (err error)
}

type telemetryWorker struct {
	providers        providers
	telemetryBuffer  *cache.SimpleCache
	benchmarksBuffer *cache.SimpleCache
	providersNetLoad *prometheus.GaugeVec
	logger           *slog.Logger
}

type Worker interface {
	UpdateTelemetry(ctx context.Context) (interval time.Duration, err error)
	UpdateBenchmarks(ctx context.Context) (interval time.Duration, err error)
}

func (w *telemetryWorker) UpdateTelemetry(ctx context.Context) (interval time.Duration, err error) {
	const (
		successInterval = 1 * time.Minute
		failureInterval = 5 * time.Second
	)

	logger := w.logger.With(slog.String("worker", "UpdateTelemetry"))
	logger.Debug("updating telemetry")

	interval = successInterval

	pubkeys, err := w.providers.GetAllProvidersPubkeys(ctx)
	if err != nil {
		interval = failureInterval
		return
	}

	items := make([]db.TelemetryUpdate, 0, len(pubkeys))
	for _, pubkey := range pubkeys {
		item, found := w.telemetryBuffer.Release(pubkey)
		if !found {
			continue
		}

		telemetryItem, ok := item.(v1.TelemetryRequest)
		if !ok {
			continue
		}

		var (
			storageGitHash  string
			providerGitHash string
		)
		if telemetryItem.GitHashes != nil {
			storageGitHash = telemetryItem.GitHashes["ton-storage"]
			providerGitHash = telemetryItem.GitHashes["ton-storage-provider"]
		}

		pings := "{}"
		if telemetryItem.Pings != nil {
			p, pErr := json.Marshal(telemetryItem.Pings)
			if pErr == nil {
				pings = string(p)
			}
		}

		items = append(items, db.TelemetryUpdate{
			PublicKey:          strings.ToLower(telemetryItem.Storage.Provider.PubKey),
			UsedProviderSpace:  telemetryItem.Storage.Provider.UsedProviderSpace,
			TotalProviderSpace: telemetryItem.Storage.Provider.TotalProviderSpace,
			StorageGitHash:     storageGitHash,
			ProviderGitHash:    providerGitHash,
			DiskName:           telemetryItem.Storage.DiskName,
			TotalDiskSpace:     telemetryItem.Storage.TotalDiskSpace,
			FreeDiskSpace:      telemetryItem.Storage.FreeDiskSpace,
			UsedDiskSpace:      telemetryItem.Storage.UsedDiskSpace,
			RAMTotal:           telemetryItem.Memory.Total,
			RAMUsage:           telemetryItem.Memory.Usage,
			RAMUsagePercent:    telemetryItem.Memory.UsagePercent,
			SwapTotal:          telemetryItem.Swap.Total,
			SwapUsage:          telemetryItem.Swap.Usage,
			SwapUsagePercent:   telemetryItem.Swap.UsagePercent,
			USysname:           telemetryItem.Uname.Sysname,
			URelease:           telemetryItem.Uname.Release,
			UVersion:           telemetryItem.Uname.Version,
			UMachine:           telemetryItem.Uname.Machine,
			CPUNumber:          telemetryItem.CPUInfo.Number,
			CPULoad:            telemetryItem.CPUInfo.CPULoad,
			CPUName:            telemetryItem.CPUInfo.CPUName,
			CPUProductName:     telemetryItem.CPUInfo.ProductName,
			CPUIsVirtual:       telemetryItem.CPUInfo.IsVirtual,
			MaxBagSizeBytes:    telemetryItem.Storage.Provider.MaxBagSizeBytes,
			Pings:              pings,
			TelemetryIP:        telemetryItem.XRealIP,
			NetLoad:            telemetryItem.NetLoad,
			NetReceived:        telemetryItem.NetReceived,
			NetSent:            telemetryItem.NetSent,
			DisksLoad:          telemetryItem.DisksLoad,
			DisksLoadPercent:   telemetryItem.DisksLoadPercent,
			IOPS:               telemetryItem.IOPS,
			PPS:                telemetryItem.PPS,
		})

		if len(telemetryItem.NetLoad) > 0 {
			w.providersNetLoad.WithLabelValues(strings.ToLower(pubkey), "total").Set(float64(telemetryItem.NetLoad[0]))
		}
		if len(telemetryItem.NetReceived) > 0 {
			w.providersNetLoad.WithLabelValues(strings.ToLower(pubkey), "received").Set(float64(telemetryItem.NetReceived[0]))
		}
		if len(telemetryItem.NetSent) > 0 {
			w.providersNetLoad.WithLabelValues(strings.ToLower(pubkey), "sent").Set(float64(telemetryItem.NetSent[0]))
		}
	}

	if len(items) == 0 {
		return
	}

	err = w.providers.UpdateTelemetry(ctx, items)
	if err != nil {
		interval = failureInterval
		return
	}

	logger.Debug("telemetry updated successfully", slog.Int("count", len(items)))

	return
}

func (w *telemetryWorker) UpdateBenchmarks(ctx context.Context) (interval time.Duration, err error) {
	const (
		successInterval = 1 * time.Minute
		failureInterval = 5 * time.Second
	)

	logger := w.logger.With(slog.String("worker", "UpdateBenchmarks"))

	interval = successInterval

	pubkeys, err := w.providers.GetAllProvidersPubkeys(ctx)
	if err != nil {
		interval = failureInterval
		return
	}

	items := make([]db.BenchmarkUpdate, 0, len(pubkeys))
	for _, pubkey := range pubkeys {
		item, found := w.benchmarksBuffer.Release(strings.ToLower(pubkey))
		if !found {
			continue
		}

		benchmarkItem, ok := item.(v1.BenchmarksRequest)
		if !ok {
			continue
		}

		disk := "{}"
		if d, jErr := json.Marshal(benchmarkItem.Disk); jErr == nil {
			disk = string(d)
		}

		var diskReadSpeed, diskWriteSpeed string
		if v, ok := benchmarkItem.Disk["qd64"]; ok {
			diskReadSpeed = v.Read
			diskWriteSpeed = v.Write
		}

		timestamp := time.Unix(benchmarkItem.Timestamp, 0)

		network := "{}"
		if ns, jErr := json.Marshal(benchmarkItem.Network); jErr == nil {
			network = string(ns)
		}

		country := benchmarkItem.Network.Client.Country
		if len(country) > 2 {
			logger.Info("Invalid country format", "coutry", country)
			country = ""
		}

		items = append(items, db.BenchmarkUpdate{
			PublicKey:          benchmarkItem.PubKey,
			Disk:               disk,
			Network:            network,
			DiskReadSpeed:      diskReadSpeed,
			DiskWriteSpeed:     diskWriteSpeed,
			SpeedtestDownload:  benchmarkItem.Network.Download,
			SpeedtestUpload:    benchmarkItem.Network.Upload,
			SpeedtestPing:      benchmarkItem.Network.Ping,
			BenchmarkTimestamp: timestamp.Format(time.RFC3339),
			Country:            country,
			ISP:                benchmarkItem.Network.Client.ISP,
		})
	}

	if len(items) == 0 {
		return
	}

	err = w.providers.UpdateBenchmarks(ctx, items)
	if err != nil {
		interval = failureInterval
		return
	}

	logger.Debug("benchmarks updated successfully", slog.Int("count", len(items)))

	return
}

func NewWorker(
	providers providers,
	telemetry *cache.SimpleCache,
	benchmarks *cache.SimpleCache,
	providersLoad *prometheus.GaugeVec,
	logger *slog.Logger,
) Worker {
	return &telemetryWorker{
		providers:        providers,
		telemetryBuffer:  telemetry,
		benchmarksBuffer: benchmarks,
		providersNetLoad: providersLoad,
		logger:           logger,
	}
}
