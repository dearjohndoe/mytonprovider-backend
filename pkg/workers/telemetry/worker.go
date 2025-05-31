package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/xssnick/tonutils-storage-provider/pkg/transport"

	"mytonprovider-backend/pkg/cache"
	v1 "mytonprovider-backend/pkg/models/api/v1"
	"mytonprovider-backend/pkg/models/db"
)

type providers interface {
	GetAllProvidersPubkeys(ctx context.Context) (pubkeys []string, err error)
	UpdateTelemetry(ctx context.Context, telemetry []db.Telemetry) (err error)
}

type telemetryWorker struct {
	providers providers
	provider  *transport.Client
	cache     *cache.SimpleCache
}

type Worker interface {
	UpdateTelemetry(ctx context.Context) (interval time.Duration, err error)
}

func (w *telemetryWorker) UpdateTelemetry(ctx context.Context) (interval time.Duration, err error) {
	const (
		successInterval = 1 * time.Minute
		failureInterval = 5 * time.Second
	)

	interval = successInterval

	pubkeys, err := w.providers.GetAllProvidersPubkeys(ctx)
	if err != nil {
		interval = failureInterval
		return
	}

	items := make([]db.Telemetry, 0, len(pubkeys))
	for _, pubkey := range pubkeys {
		item, found := w.cache.Release(pubkey)
		if !found {
			continue
		}

		telemetryItem, ok := item.(*v1.TelemetryRequest)
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

		// TODO: separate
		benchmarks := "{}"
		b, bErr := json.Marshal(telemetryItem.Benchmark)
		if bErr == nil {
			benchmarks = string(b)
		}

		pings := "{}"
		if telemetryItem.Pings != nil {
			p, pErr := json.Marshal(telemetryItem.Pings)
			if pErr == nil {
				pings = string(p)
			}
		}

		items = append(items, db.Telemetry{
			PublicKey:               strings.ToLower(telemetryItem.Storage.Provider.PubKey),
			UsedProviderSpace:       telemetryItem.Storage.Provider.UsedProviderSpace,
			TotalProviderSpace:      telemetryItem.Storage.Provider.TotalProviderSpace,
			StorageGitHash:          storageGitHash,
			ProviderGitHash:         providerGitHash,
			Country:                 "",
			ISP:                     "",
			DiskName:                telemetryItem.Storage.DiskName,
			TotalDiskSpace:          telemetryItem.Storage.TotalDiskSpace,
			FreeDiskSpace:           telemetryItem.Storage.FreeDiskSpace,
			UsedDiskSpace:           telemetryItem.Storage.UsedDiskSpace,
			RAMTotal:                telemetryItem.Memory.Total,
			RAMFree:                 telemetryItem.Memory.Usage,
			RAMUsagePercent:         telemetryItem.Memory.UsagePercent,
			SwapTotal:               telemetryItem.Swap.Total,
			SwapFree:                telemetryItem.Swap.Usage,
			SwapUsagePercent:        telemetryItem.Swap.UsagePercent,
			USysname:                telemetryItem.Uname.Sysname,
			URelease:                telemetryItem.Uname.Release,
			UVersion:                telemetryItem.Uname.Version,
			UMachine:                telemetryItem.Uname.Machine,
			CPUNumber:               telemetryItem.CPUInfo.Number,
			CPULoad:                 telemetryItem.CPUInfo.CPULoad,
			CPUName:                 telemetryItem.CPUInfo.CPUName,
			CPUProductName:          telemetryItem.CPUInfo.ProductName,
			CPUIsVirtual:            telemetryItem.CPUInfo.IsVirtual,
			BenchmarkDiskReadSpeed:  0,
			BenchmarkDiskWriteSpeed: 0,
			BenchmarkRocksOps:       0,
			SpeedtestDownloadSpeed:  0,
			SpeedtestUploadSpeed:    0,
			SpeedtestPing:           0.0,
			Pings:                   pings,
			Benchmarks:              benchmarks,
		})
	}

	if len(items) == 0 {
		return
	}

	fmt.Println("Updating telemetry", items)

	err = w.providers.UpdateTelemetry(ctx, items)
	if err != nil {
		interval = failureInterval
		return
	}

	return
}

func NewWorker(
	providers providers,
	telemetry *cache.SimpleCache,
) Worker {
	return &telemetryWorker{
		providers: providers,
		cache:     telemetry,
	}
}
