package providersmaster

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/xssnick/tonutils-go/adnl/dht"
	"github.com/xssnick/tonutils-storage-provider/pkg/transport"

	"mytonprovider-coordinator/internal/models/db"
)

var (
	ErrNoStorageContracts           = errors.New("no storage contracts available")
	ErrNoResolvedProviderEndpoints  = errors.New("no providers with resolved endpoints")
	ErrNoValidProvidersForRunChecks = errors.New("no valid providers collected for RunChecksRequest")
)

type RunChecksRequestPayload struct {
	JobID     string                 `json:"jobId"`
	Providers []ProviderBatchPayload `json:"providers"`
	Timeouts  CheckTimeoutsPayload   `json:"timeouts"`
}

type ProviderBatchPayload struct {
	ProviderPubkey  string               `json:"providerPubkey"`
	ProviderAddress string               `json:"providerAddress"`
	StorageEndpoint EndpointPayload      `json:"storageEndpoint"`
	Contracts       []ContractRefPayload `json:"contracts"`
}

type EndpointPayload struct {
	IP         string `json:"ip"`
	Port       int32  `json:"port"`
	ADNLPubkey []byte `json:"adnlPubkey"`
}

type ContractRefPayload struct {
	ContractAddress string `json:"contractAddress"`
	BagID           string `json:"bagId"`
}

type CheckTimeoutsPayload struct {
	PingMs  uint32 `json:"pingMs"`
	RldpMs  uint32 `json:"rldpMs"`
	TotalMs uint32 `json:"totalMs"`
}

type RequestBuilder interface {
	BuildRunChecksRequest(ctx context.Context, jobID string, providerLimit int, timeouts CheckTimeoutsPayload) (*RunChecksRequestPayload, error)
}

func NewRequestBuilder(
	providers providers,
	system system,
	ton ton,
	providerClient *transport.Client,
	dhtClient *dht.Client,
	ipinfo ipclient,
	logger *slog.Logger,
) RequestBuilder {
	_, prv, err := ed25519.GenerateKey(nil)
	if err != nil {
		logger.Error("failed to generate ed25519 key", "error", err)
		return nil
	}

	return &providersMasterWorker{
		providers:      providers,
		system:         system,
		ton:            ton,
		prv:            prv,
		providerClient: providerClient,
		dhtClient:      dhtClient,
		ipinfo:         ipinfo,
		logger:         logger,
	}
}

func (w *providersMasterWorker) BuildRunChecksRequest(ctx context.Context, jobID string, providerLimit int, timeouts CheckTimeoutsPayload) (*RunChecksRequestPayload, error) {
	if providerLimit < 0 {
		return nil, fmt.Errorf("providerLimit must be >= 0")
	}

	storageContracts, err := w.providers.GetStorageContracts(ctx)
	if err != nil {
		return nil, fmt.Errorf("get storage contracts: %w", err)
	}
	if len(storageContracts) == 0 {
		return nil, ErrNoStorageContracts
	}

	availableProvidersIPs, err := w.resolveProvidersIPsNoPersist(ctx, storageContracts)
	if err != nil {
		return nil, fmt.Errorf("resolve provider IPs: %w", err)
	}
	if len(availableProvidersIPs) == 0 {
		return nil, ErrNoResolvedProviderEndpoints
	}

	contractsByProvider := make(map[string][]db.ContractToProviderRelation)
	for _, sc := range storageContracts {
		if _, ok := availableProvidersIPs[sc.ProviderPublicKey]; !ok {
			continue
		}
		contractsByProvider[sc.ProviderPublicKey] = append(contractsByProvider[sc.ProviderPublicKey], sc)
	}

	pubkeys := make([]string, 0, len(contractsByProvider))
	for pk := range contractsByProvider {
		pubkeys = append(pubkeys, pk)
	}
	sort.Strings(pubkeys)

	if providerLimit > 0 && providerLimit < len(pubkeys) {
		pubkeys = pubkeys[:providerLimit]
	}

	if jobID == "" {
		jobID = fmt.Sprintf("dump-%d", time.Now().Unix())
	}
	if timeouts.TotalMs == 0 {
		timeouts = CheckTimeoutsPayload{
			PingMs:  7000,
			RldpMs:  10000,
			TotalMs: 30000,
		}
	}

	resp := &RunChecksRequestPayload{
		JobID:     jobID,
		Providers: make([]ProviderBatchPayload, 0, len(pubkeys)),
		Timeouts:  timeouts,
	}

	for _, pk := range pubkeys {
		ip := availableProvidersIPs[pk]
		if strings.TrimSpace(ip.Storage.IP) == "" || ip.Storage.Port <= 0 || len(ip.Storage.PublicKey) != ed25519.PublicKeySize {
			continue
		}

		contracts := contractsByProvider[pk]
		if len(contracts) == 0 {
			continue
		}

		contractRefs := make([]ContractRefPayload, 0, len(contracts))
		for _, c := range contracts {
			contractRefs = append(contractRefs, ContractRefPayload{
				ContractAddress: c.Address,
				BagID:           c.BagID,
			})
		}

		resp.Providers = append(resp.Providers, ProviderBatchPayload{
			ProviderPubkey:  pk,
			ProviderAddress: contracts[0].ProviderAddress,
			StorageEndpoint: EndpointPayload{
				IP:         ip.Storage.IP,
				Port:       ip.Storage.Port,
				ADNLPubkey: ip.Storage.PublicKey,
			},
			Contracts: contractRefs,
		})
	}

	if len(resp.Providers) == 0 {
		return nil, ErrNoValidProvidersForRunChecks
	}

	return resp, nil
}

func (w *providersMasterWorker) resolveProvidersIPsNoPersist(ctx context.Context, storageContracts []db.ContractToProviderRelation) (map[string]db.ProviderIP, error) {
	log := w.logger.With(slog.String("worker", "DumpRunChecks"), slog.String("function", "resolveProvidersIPsNoPersist"))

	uniqueProviders := make(map[string]db.ContractToProviderRelation)
	for _, sc := range storageContracts {
		if _, exists := uniqueProviders[sc.ProviderPublicKey]; !exists {
			uniqueProviders[sc.ProviderPublicKey] = sc
		}
	}

	availableProvidersIPs := make(map[string]db.ProviderIP, len(uniqueProviders))
	notFoundIPs := make([]string, 0)
	semaphore := make(chan struct{}, maxConcurrentProviderChecks)

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, sc := range uniqueProviders {
		wg.Add(1)
		go func(contract db.ContractToProviderRelation) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			providerIPs, pErr := w.findProviderIPs(ctx, contract, log)

			mu.Lock()
			defer mu.Unlock()
			if pErr != nil {
				notFoundIPs = append(notFoundIPs, contract.ProviderPublicKey)
			}
			availableProvidersIPs[contract.ProviderPublicKey] = providerIPs
		}(sc)
	}

	wg.Wait()

	for _, pk := range notFoundIPs {
		ip := availableProvidersIPs[pk]
		if ip.Provider.IP == "" {
			delete(availableProvidersIPs, pk)
			continue
		}

		providerContracts := make([]db.ContractToProviderRelation, 0)
		for _, sc := range storageContracts {
			if sc.ProviderPublicKey == pk {
				providerContracts = append(providerContracts, sc)
			}
		}
		if len(providerContracts) == 0 {
			delete(availableProvidersIPs, pk)
			continue
		}

		storageIP, err := w.findStorageIPOverlay(ctx, ip.Provider.IP, providerContracts, ip.Storage, log)
		if err != nil {
			delete(availableProvidersIPs, pk)
			continue
		}

		ip.Storage = storageIP
		availableProvidersIPs[pk] = ip
	}

	return availableProvidersIPs, nil
}
