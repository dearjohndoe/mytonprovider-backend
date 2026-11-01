package providersmaster

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/adnl"
	adnladdress "github.com/xssnick/tonutils-go/adnl/address"
	"github.com/xssnick/tonutils-go/adnl/dht"
	"github.com/xssnick/tonutils-go/adnl/keys"
	"github.com/xssnick/tonutils-go/adnl/overlay"
	"github.com/xssnick/tonutils-go/adnl/rldp"
	"github.com/xssnick/tonutils-go/tl"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/tvm/cell"
	"github.com/xssnick/tonutils-storage-provider/pkg/transport"

	providerchecksv1 "github.com/rudolfkova/mytonprovider-backend/contracts/gen/go/providerchecks/v1"
	"mytonprovider-coordinator/internal/clients/agentrpc"
	"mytonprovider-coordinator/internal/clients/ifconfig"
	tonclient "mytonprovider-coordinator/internal/clients/ton"
	"mytonprovider-coordinator/internal/constants"
	"mytonprovider-coordinator/internal/models/db"
	"mytonprovider-coordinator/internal/pipelineevents"
	"mytonprovider-coordinator/internal/tonstorage"
	"mytonprovider-coordinator/internal/utils"
)

const (
	lastLTKey                     = "masterWalletLastLT"
	prefix                        = "tsp-"
	storageRewardWithdrawalOpCode = 0xa91baf56
	maxConcurrentProviderChecks   = 30
	maxConcurrentBagChecks        = 30
	fakeSize                      = 1
	verifyStorageRetries          = 3

	// Timeout durations
	providerResponseTimeout = 14 * time.Second
	dhtTimeout              = 14 * time.Second
	pingTimeout             = 7 * time.Second
	rlQueryTimeout          = 10 * time.Second
	getTxTimeout            = 20 * time.Second
	ipInfoTimeout           = 10 * time.Second
	ipInfoSleepDuration     = 1 * time.Second

	defaultEndpointStaleTTL                = 12 * time.Hour
	defaultEndpointFullRefreshInterval     = 6 * time.Hour
	defaultEndpointFullRefreshFailInterval = 30 * time.Second

	defaultStoreProofInterval   = 60 * time.Minute
	storeProofFailureInterval   = 15 * time.Second
)

type providers interface {
	GetAllProvidersPubkeys(ctx context.Context) (pubkeys []string, err error)
	GetAllProvidersWallets(ctx context.Context) (wallets []db.ProviderWallet, err error)
	UpdateProvidersLT(ctx context.Context, providers []db.ProviderWalletLT) (err error)
	AddStorageContracts(ctx context.Context, contracts []db.StorageContract) (err error)
	GetStorageContracts(ctx context.Context) (contracts []db.ContractToProviderRelation, err error)
	UpdateRejectedStorageContracts(ctx context.Context, storageContracts []db.ContractToProviderRelation) (err error)
	AddProviders(ctx context.Context, providers []db.ProviderCreate) (err error)
	UpdateProvidersIPs(ctx context.Context, ips []db.ProviderIP) (err error)
	UpdateProviders(ctx context.Context, providers []db.ProviderUpdate) (err error)
	AddStatuses(ctx context.Context, providers []db.ProviderStatusUpdate) (err error)
	UpdateContractProofsChecks(ctx context.Context, contractsProofs []db.ContractProofsCheck) (err error)
	UpdateStatuses(ctx context.Context) (err error)
	UpdateUptime(ctx context.Context) (err error)
	UpdateRating(ctx context.Context) (err error)
	GetProvidersIPs(ctx context.Context) (ips []db.ProviderIP, err error)
	GetProvidersEndpointState(ctx context.Context, pubkeys []string) (rows []db.ProviderEndpointState, err error)
	UpdateProvidersIPInfo(ctx context.Context, ips []db.ProviderIPInfo) (err error)

	InsertProviderPipelineEvents(ctx context.Context, events []db.ProviderPipelineEvent) (err error)
	InsertBagPipelineEvents(ctx context.Context, events []db.BagPipelineEvent) (err error)
	GetLastProviderPipelineEventStatus(ctx context.Context, pubkeys []string) (statusByPubkey map[string]db.PipelineEventStatus, err error)
}

type system interface {
	SetParam(ctx context.Context, key string, value string) (err error)
	GetParam(ctx context.Context, key string) (value string, err error)
}

type ton interface {
	GetTransactions(ctx context.Context, addr string, lastProcessedLT uint64) (tx []*tonclient.Transaction, err error)
	GetStorageContractsInfo(ctx context.Context, addrs []string) (contracts []tonclient.StorageContract, err error)
	GetProvidersInfo(ctx context.Context, addrs []string) (contractsProviders []tonclient.StorageContractProviders, err error)
}

type ipclient interface {
	GetIPInfo(ctx context.Context, ip string) (conf *ifconfig.Info, err error)
}

type agentclient interface {
	RunChecksAll(ctx context.Context, req *providerchecksv1.RunChecksRequest) ([]agentrpc.RunChecksResult, []agentrpc.AgentCallError)
	RunChecks(ctx context.Context, endpoint string, req *providerchecksv1.RunChecksRequest) (agentrpc.RunChecksResult, error)
	RunStorageRatesAll(ctx context.Context, req *providerchecksv1.RunStorageRatesRequest) ([]agentrpc.RunStorageRatesResult, []agentrpc.AgentCallError)
	Endpoints() []string
	AgentCount() int
}

type RunChecksTimeouts struct {
	PingMs              uint32
	RldpMs              uint32
	TotalMs             uint32
	StorageRatesQueryMs uint32
}

type EndpointRefreshConfig struct {
	StaleTTL            time.Duration
	FullRefreshInterval time.Duration
	FailInterval        time.Duration
}

type StoreProofConfig struct {
	Interval   time.Duration
	RetryDelay time.Duration
}

type providersMasterWorker struct {
	providers      providers
	system         system
	ton            ton
	ipinfo         ipclient
	prv            ed25519.PrivateKey
	providerClient *transport.Client
	agentClient    agentclient
	dhtClient      *dht.Client
	masterAddr     string
	batchSize      uint32
	timeouts       RunChecksTimeouts
	endpointCfg    EndpointRefreshConfig
	storeProofCfg  StoreProofConfig
	logger         *slog.Logger
	lastIPsMu      sync.RWMutex
	lastKnownIPs   map[string]db.ProviderIP

	providerAgentMu      sync.Mutex
	providerAgentSuccess map[string]string // providerPubkey -> endpoint of last Valid
}

type Worker interface {
	CollectNewProviders(ctx context.Context) (interval time.Duration, err error)
	UpdateKnownProviders(ctx context.Context) (interval time.Duration, err error)
	CollectProvidersNewStorageContracts(ctx context.Context) (interval time.Duration, err error)
	StoreProof(ctx context.Context) (interval time.Duration, err error)
	UpdateUptime(ctx context.Context) (interval time.Duration, err error)
	UpdateRating(ctx context.Context) (interval time.Duration, err error)
	UpdateIPInfo(ctx context.Context) (interval time.Duration, err error)
	RefreshEndpointsFull(ctx context.Context) (interval time.Duration, err error)
}

func (w *providersMasterWorker) CollectNewProviders(ctx context.Context) (interval time.Duration, err error) {
	const (
		successInterval = 1 * time.Minute
		failureInterval = 5 * time.Second
	)

	log := w.logger.With("worker", "CollectNewProviders")
	log.Debug("collecting new providers")

	interval = successInterval

	lv, err := w.system.GetParam(ctx, lastLTKey)
	if err != nil {
		interval = failureInterval
		return
	}

	// ignore error. Zero will scann all transactions that lite server return, so we ok
	lastProcessedLT, _ := strconv.ParseInt(lv, 10, 64)

	p, err := w.providers.GetAllProvidersPubkeys(ctx)
	if err != nil {
		interval = failureInterval
		return
	}

	knownProviders := make(map[string]struct{}, len(p))
	for _, pubkey := range p {
		knownProviders[strings.ToLower(pubkey)] = struct{}{}
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, getTxTimeout)
	defer cancel()

	txs, err := w.ton.GetTransactions(timeoutCtx, w.masterAddr, uint64(lastProcessedLT))
	if err != nil {
		interval = failureInterval
		return
	}

	uniqueProviders := make(map[string]db.ProviderCreate)
	biggestLT := uint64(lastProcessedLT)
	for i := range txs {
		if txs[i].LT <= uint64(lastProcessedLT) {
			continue
		}

		if biggestLT < txs[i].LT {
			biggestLT = txs[i].LT
		}

		pos := strings.Index(txs[i].Message, prefix)
		if pos < 0 {
			continue
		}

		pos += len(prefix)
		if pos >= len(txs[i].Message) {
			continue
		}

		pubkey := strings.ToLower(txs[i].Message[pos:])

		if len(pubkey) != 64 {
			continue
		}

		if _, ok := knownProviders[pubkey]; ok {
			continue
		}

		prv, err := hex.DecodeString(pubkey)
		if err != nil || len(prv) != 32 {
			continue
		}

		uniqueProviders[pubkey] = db.ProviderCreate{
			Pubkey:       pubkey,
			Address:      txs[i].From,
			RegisteredAt: txs[i].CreatedAt,
		}
	}

	if len(uniqueProviders) == 0 {
		return
	}

	if biggestLT > uint64(lastProcessedLT) {
		errP := w.system.SetParam(ctx, lastLTKey, strconv.FormatUint(biggestLT, 10))
		if errP != nil {
			log.Error("cannot update last processed LT for master wallet", "error", errP.Error())
		}
	}

	providersInit := make([]db.ProviderCreate, 0, len(uniqueProviders))
	for _, provider := range uniqueProviders {
		providersInit = append(providersInit, provider)
	}

	err = w.providers.AddProviders(ctx, providersInit)
	if err != nil {
		interval = failureInterval
		return
	}

	log.Info("successfully collected new providers", "count", len(providersInit))

	return
}

func (w *providersMasterWorker) UpdateKnownProviders(ctx context.Context) (interval time.Duration, err error) {
	const (
		successInterval = 1 * time.Minute
		failureInterval = 5 * time.Second
	)

	log := w.logger.With(slog.String("worker", "UpdateKnownProviders"))
	log.Debug("updating known providers")

	interval = successInterval

	p, err := w.providers.GetAllProvidersPubkeys(ctx)
	if err != nil {
		interval = failureInterval
		return
	}

	if len(p) == 0 {
		return
	}

	if w.agentClient == nil || w.agentClient.AgentCount() == 0 {
		log.Error("no configured agent clients for UpdateKnownProviders")
		interval = failureInterval
		return
	}

	req := &providerchecksv1.RunStorageRatesRequest{
		JobId:           fmt.Sprintf("update-known-providers-%d", time.Now().Unix()),
		ProviderPubkeys: p,
		QuerySize:       fakeSize,
		Timeouts: &providerchecksv1.StorageRatesTimeouts{
			QueryTimeoutMs: w.timeouts.StorageRatesQueryMs,
		},
	}
	responses, callErrs := w.agentClient.RunStorageRatesAll(ctx, req)
	for _, callErr := range callErrs {
		log.Warn("RunStorageRates failed for agent", "endpoint", callErr.Endpoint, "error", callErr.Err)
	}
	if len(responses) == 0 {
		log.Error("all agents are unavailable for RunStorageRates", "agents_total", w.agentClient.AgentCount())
		interval = failureInterval
		return
	}

	merged := mergeStorageRatesResponses(p, responses)
	providersInfo := make([]db.ProviderUpdate, 0, len(p))
	providersStatuses := make([]db.ProviderStatusUpdate, 0, len(p))
	for _, pubkey := range p {
		rates, ok := merged[pubkey]
		if !ok || rates == nil || !rates.GetOk() {
			providersStatuses = append(providersStatuses, db.ProviderStatusUpdate{
				Pubkey:   pubkey,
				IsOnline: false,
			})
			continue
		}

		providersStatuses = append(providersStatuses, db.ProviderStatusUpdate{
			Pubkey:   pubkey,
			IsOnline: true,
		})

		providersInfo = append(providersInfo, db.ProviderUpdate{
			Pubkey:       pubkey,
			RatePerMBDay: new(big.Int).SetBytes(rates.RatePerMbDay).Int64(),
			MinBounty:    new(big.Int).SetBytes(rates.MinBounty).Int64(),
			MinSpan:      rates.MinSpan,
			MaxSpan:      rates.MaxSpan,
		})
	}

	err = w.providers.AddStatuses(ctx, providersStatuses)
	if err != nil {
		interval = failureInterval
		return
	}

	err = w.providers.UpdateProviders(ctx, providersInfo)
	if err != nil {
		interval = failureInterval
		return
	}

	log.Info("successfully updated known providers", "active", len(providersInfo))

	return
}

func (w *providersMasterWorker) CollectProvidersNewStorageContracts(ctx context.Context) (interval time.Duration, err error) {
	const (
		successInterval = 60 * time.Minute
		failureInterval = 15 * time.Second
	)

	log := w.logger.With("worker", "ProvidersContracts")
	log.Debug("collect new providers contracts")

	interval = successInterval

	providersWallets, err := w.providers.GetAllProvidersWallets(ctx)
	if err != nil {
		interval = failureInterval
		return
	}

	providersToUpdate := make([]db.ProviderWalletLT, 0, len(providersWallets))
	storageContracts := make(map[string]db.StorageContract)

	wg := sync.WaitGroup{}
	smu := sync.Mutex{}
	pmu := sync.Mutex{}

	wg.Add(len(providersWallets))
	for _, provider := range providersWallets {
		go func(ctx context.Context, provider db.ProviderWallet) {
			defer wg.Done()

			var lastLT uint64
			sc, lastLT, err := w.scanProviderTransactions(ctx, provider)
			if err != nil {
				log.Error("failed to scan provider transactions", "address", provider.Address, "error", err)
				return
			}

			if len(sc) > 0 {
				smu.Lock()
				for src, tx := range sc {
					if v, ok := storageContracts[src]; ok {
						for p := range tx.ProvidersAddresses {
							v.ProvidersAddresses[p] = struct{}{}
						}
						if v.LastLT < tx.LastLT {
							v.LastLT = tx.LastLT
						}
						storageContracts[src] = v
					} else {
						storageContracts[src] = tx
					}
				}
				smu.Unlock()
			}

			if lastLT != provider.LT {
				pmu.Lock()
				providersToUpdate = append(providersToUpdate, db.ProviderWalletLT{
					PubKey: provider.PubKey,
					LT:     lastLT,
				})
				pmu.Unlock()
			}
		}(ctx, provider)
	}

	wg.Wait()

	if len(storageContracts) == 0 {
		return
	}

	// Collect more info about storage contracts
	contractsAdresses := make([]string, 0, len(storageContracts))
	for address := range storageContracts {
		contractsAdresses = append(contractsAdresses, address)
	}

	contractsInfo, err := w.ton.GetStorageContractsInfo(ctx, contractsAdresses)
	if err != nil {
		log.Error("failed to get storage contracts info", "error", err)
		interval = failureInterval
		return
	}

	newContracts := make([]db.StorageContract, 0, len(contractsInfo))
	for _, contract := range contractsInfo {
		sc, ok := storageContracts[contract.Address]
		if !ok {
			log.Error("storage contract not found in scanned transactions", "address", contract.Address)
			continue
		}

		newContracts = append(newContracts, db.StorageContract{
			ProvidersAddresses: sc.ProvidersAddresses,
			Address:            contract.Address,
			BagID:              contract.BagID,
			OwnerAddr:          contract.OwnerAddr,
			Size:               contract.Size,
			ChunkSize:          contract.ChunkSize,
			LastLT:             sc.LastLT,
		})
	}

	err = w.providers.UpdateProvidersLT(ctx, providersToUpdate)
	if err != nil {
		log.Error("failed to update providers wallets lt", "error", err)
		interval = failureInterval
		return
	}

	err = w.providers.AddStorageContracts(ctx, newContracts)
	if err != nil {
		log.Error("failed to add storage contracts", "error", err)
		interval = failureInterval
		return
	}

	log.Info("successfully collected new storage contracts", "count", len(newContracts))

	return
}

func (w *providersMasterWorker) StoreProof(ctx context.Context) (interval time.Duration, err error) {
	started := time.Now()
	cycleInterval := w.storeProofCfg.Interval
	if cycleInterval <= 0 {
		cycleInterval = defaultStoreProofInterval
	}
	nextFullAt := started.Add(cycleInterval)
	remain := func() time.Duration {
		d := time.Until(nextFullAt)
		if d < 0 {
			return 0
		}
		return d
	}

	log := w.logger.With(slog.String("worker", "StoreProof"))
	log.Debug("checking storage proofs")

	interval = remain()

	runID := fmt.Sprintf("storeproof-%d", started.Unix())
	recorder := pipelineevents.NewRecorder(w.providers, w.logger, runID, "StoreProof")

	storageContracts, err := w.providers.GetStorageContracts(ctx)
	if err != nil {
		log.Error("failed to get storage contracts", "error", err)
		interval = storeProofFailureInterval
		return
	}

	if _, refreshErr := w.refreshProvidersEndpoints(ctx, storageContracts, true, recorder); refreshErr != nil {
		log.Warn("stale-first endpoint refresh failed before StoreProof", "error", refreshErr)
	}

	storageContracts, err = w.updateRejectedContracts(ctx, storageContracts)
	if err != nil {
		interval = storeProofFailureInterval
		return
	}

	availableProvidersIPs, err := w.updateProvidersIPs(ctx, storageContracts, recorder)
	if err != nil {
		interval = storeProofFailureInterval
		return
	}

	mainResults, mainAssignment, err := w.runStoreProofChecks(ctx, runID, storageContracts, availableProvidersIPs, recorder, nil)
	if err != nil {
		interval = storeProofFailureInterval
		recorder.Flush(ctx)
		return
	}

	toCommit, pendingKeys := selectMainProofsCommit(mainResults)
	if len(toCommit) > 0 {
		if err = w.providers.UpdateContractProofsChecks(ctx, toCommit); err != nil {
			log.Error("failed to update contract proofs checks after main pass", "error", err)
			interval = storeProofFailureInterval
			recorder.Flush(ctx)
			return
		}
	}

	if err = w.providers.UpdateStatuses(ctx); err != nil {
		log.Error("failed to update provider statuses after main pass", "error", err)
		interval = storeProofFailureInterval
		recorder.Flush(ctx)
		return
	}
	recorder.Flush(ctx)

	log.Info(
		"StoreProof main pass committed",
		"committed", len(toCommit),
		"pending_retry", len(pendingKeys),
		"next_full_at", nextFullAt,
	)

	retryDelay := w.storeProofCfg.RetryDelay
	retryDeadline := nextFullAt.Add(-storeProofRetryBuffer)

	shouldRetry := len(pendingKeys) > 0 && retryDelay > 0 && time.Now().Before(retryDeadline)
	if !shouldRetry {
		if len(pendingKeys) > 0 {
			finalCommit := selectFinalProofsCommit(mainResults, pendingKeys, nil)
			if len(finalCommit) > 0 {
				if err = w.providers.UpdateContractProofsChecks(ctx, finalCommit); err != nil {
					log.Error("failed to commit deferred proof fails without retry", "error", err)
					interval = storeProofFailureInterval
					return
				}
				if err = w.providers.UpdateStatuses(ctx); err != nil {
					log.Error("failed to update provider statuses after deferred commit", "error", err)
					interval = storeProofFailureInterval
					return
				}
			}
			log.Info("StoreProof skipped retry, committed deferred fails", "count", len(finalCommit), "retry_delay", retryDelay)
		}
		interval = remain()
		return
	}

	sleepUntil := time.Now().Add(retryDelay)
	if sleepUntil.After(retryDeadline) {
		sleepUntil = retryDeadline
	}
	if err = waitUntil(ctx, sleepUntil); err != nil {
		finalCommit := selectFinalProofsCommit(mainResults, pendingKeys, nil)
		if len(finalCommit) > 0 {
			commitCtx := context.WithoutCancel(ctx)
			if cErr := w.providers.UpdateContractProofsChecks(commitCtx, finalCommit); cErr != nil {
				log.Error("failed to commit deferred proof fails after cancel", "error", cErr)
			} else if sErr := w.providers.UpdateStatuses(commitCtx); sErr != nil {
				log.Error("failed to update statuses after cancel", "error", sErr)
			}
		}
		interval = remain()
		return
	}

	if !time.Now().Before(retryDeadline) {
		finalCommit := selectFinalProofsCommit(mainResults, pendingKeys, nil)
		if len(finalCommit) > 0 {
			if err = w.providers.UpdateContractProofsChecks(ctx, finalCommit); err != nil {
				log.Error("failed to commit deferred proof fails after delay", "error", err)
				interval = storeProofFailureInterval
				return
			}
			if err = w.providers.UpdateStatuses(ctx); err != nil {
				log.Error("failed to update provider statuses after delay", "error", err)
				interval = storeProofFailureInterval
				return
			}
		}
		interval = remain()
		return
	}

	pending := pendingContracts(storageContracts, pendingKeys)
	retryRecorder := pipelineevents.NewRecorder(w.providers, w.logger, runID, "StoreProof")
	refreshed, refreshErr := w.refreshProvidersEndpoints(ctx, pending, false, retryRecorder)
	if refreshErr != nil {
		log.Warn("force endpoint refresh before StoreProof retry failed", "error", refreshErr)
	}
	for pk, ip := range refreshed {
		availableProvidersIPs[pk] = ip
	}

	retryCtx, cancelRetry := context.WithDeadline(ctx, retryDeadline)
	defer cancelRetry()

	retryResults, _, retryErr := w.runStoreProofChecks(retryCtx, runID+"-retry", pending, availableProvidersIPs, retryRecorder, mainAssignment)
	if retryErr != nil {
		log.Warn("StoreProof retry RunChecks failed", "error", retryErr)
		retryResults = nil
	}

	finalCommit := selectFinalProofsCommit(mainResults, pendingKeys, retryResults)
	if len(finalCommit) > 0 {
		if err = w.providers.UpdateContractProofsChecks(ctx, finalCommit); err != nil {
			log.Error("failed to update contract proofs checks after retry", "error", err)
			interval = storeProofFailureInterval
			retryRecorder.Flush(ctx)
			return
		}
	}
	if err = w.providers.UpdateStatuses(ctx); err != nil {
		log.Error("failed to update provider statuses after retry", "error", err)
		interval = storeProofFailureInterval
		retryRecorder.Flush(ctx)
		return
	}
	retryRecorder.Flush(ctx)

	log.Info(
		"StoreProof retry pass committed",
		"committed", len(finalCommit),
		"pending_was", len(pendingKeys),
		"retry_error", retryErr != nil,
	)

	interval = remain()
	return
}

func waitUntil(ctx context.Context, until time.Time) error {
	d := time.Until(until)
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func (w *providersMasterWorker) UpdateUptime(ctx context.Context) (interval time.Duration, err error) {
	const (
		successInterval = 5 * time.Minute
		failureInterval = 5 * time.Second
	)

	log := w.logger.With(slog.String("worker", "UpdateUptime"))
	log.Debug("updating provider uptime")

	interval = successInterval

	err = w.providers.UpdateUptime(ctx)
	if err != nil {
		interval = failureInterval
		return
	}

	return
}

func (w *providersMasterWorker) UpdateRating(ctx context.Context) (interval time.Duration, err error) {
	const (
		successInterval = 5 * time.Minute
		failureInterval = 5 * time.Second
	)

	log := w.logger.With(slog.String("worker", "UpdateRating"))
	log.Debug("updating provider ratings")

	interval = successInterval

	err = w.providers.UpdateRating(ctx)
	if err != nil {
		interval = failureInterval
		return
	}

	return
}

func (w *providersMasterWorker) UpdateIPInfo(ctx context.Context) (interval time.Duration, err error) {
	const (
		successInterval = 240 * time.Minute
		failureInterval = 30 * time.Second
	)

	log := w.logger.With(slog.String("worker", "UpdateIPInfo"))
	log.Debug("updating provider IP info")

	interval = failureInterval

	ips, err := w.providers.GetProvidersIPs(ctx)
	if err != nil {
		log.Error("failed to get provider IPs", "error", err)
		return
	}

	if len(ips) == 0 {
		log.Info("no provider IPs to update")
		interval = successInterval
		return
	}

	ipsInfo := make([]db.ProviderIPInfo, 0, len(ips))
	for _, ip := range ips {
		time.Sleep(ipInfoSleepDuration)

		ipErr := func() error {
			timeoutCtx, cancel := context.WithTimeout(ctx, ipInfoTimeout)
			defer cancel()

			info, err := w.ipinfo.GetIPInfo(timeoutCtx, ip.Provider.IP)
			if err != nil {
				return fmt.Errorf("failed to get IP info: %w", err)
			}

			s, err := json.Marshal(info)
			if err != nil {
				return fmt.Errorf("failed to marshal IP info: %w, ip: %s, info: %s", err, ip.Provider.IP, info)
			}

			ipsInfo = append(ipsInfo, db.ProviderIPInfo{
				PublicKey: ip.PublicKey,
				IPInfo:    string(s),
			})

			return nil
		}()
		if ipErr != nil {
			log.Error(ipErr.Error())
			continue
		}
	}

	err = w.providers.UpdateProvidersIPInfo(ctx, ipsInfo)
	if err != nil {
		log.Error("failed to update provider IP info", "error", err)
		interval = failureInterval
		return
	}

	interval = successInterval

	return
}

// runStoreProofChecks shards RunChecks across agents, merges results, and records pipeline events.
// excludeByProvider maps providerPubkey -> endpoint to avoid on retry when another agent exists.
// assignment is providerPubkey -> endpoint used for this pass.
// It does not write public reason to the database.
func (w *providersMasterWorker) runStoreProofChecks(
	ctx context.Context,
	jobID string,
	storageContracts []db.ContractToProviderRelation,
	availableProvidersIPs map[string]db.ProviderIP,
	recorder *pipelineevents.Recorder,
	excludeByProvider map[string]string,
) (results map[string]mergedContractCheck, assignment map[string]string, err error) {
	log := w.logger.With(slog.String("worker", "StoreProof"), slog.String("function", "runStoreProofChecks"), slog.String("job_id", jobID))
	if w.agentClient == nil || w.agentClient.AgentCount() == 0 {
		return nil, nil, fmt.Errorf("no configured agent clients for RunChecks")
	}
	if len(storageContracts) == 0 {
		return map[string]mergedContractCheck{}, map[string]string{}, nil
	}

	req, providerCount, checkedKeys := w.buildRunChecksRPCRequest(jobID, storageContracts, availableProvidersIPs)
	if providerCount == 0 {
		if recorder != nil {
			for _, sc := range storageContracts {
				if _, ok := checkedKeys[contractRelationKey(sc)]; !ok {
					recorder.RecordBagEndpointUnresolved(sc)
				}
			}
		}
		results = buildBagCheckResults(storageContracts, checkedKeys, nil)
		log.Warn("no providers with resolved endpoints for RunChecks", "contracts", len(storageContracts))
		return results, map[string]string{}, nil
	}

	endpoints := w.agentClient.Endpoints()
	if len(endpoints) == 0 {
		return nil, nil, fmt.Errorf("no configured agent endpoints for RunChecks")
	}

	sticky := w.snapshotProviderAgentSuccess()
	shard := assignProvidersToAgents(req.Providers, endpoints, sticky, excludeByProvider)
	assignment = shard.byProvider

	responses, callErrs := w.fanoutRunChecks(ctx, req, shard.byEndpoint)
	for _, callErr := range callErrs {
		log.Warn("RunChecks failed for agent", "endpoint", callErr.Endpoint, "error", callErr.Err)
	}
	if len(responses) == 0 {
		return nil, assignment, fmt.Errorf("all agents are unavailable for RunChecks")
	}

	merged, valid := mergeRunChecksResponses(storageContracts, responses)
	results = buildBagCheckResults(storageContracts, checkedKeys, merged)
	w.updateProviderAgentSuccess(storageContracts, results, assignment)

	if recorder != nil {
		for _, sc := range storageContracts {
			key := contractRelationKey(sc)
			row, ok := results[key]
			if !ok {
				continue
			}
			recorder.RecordBagTransition(sc, row.Reason, row.Stage, row.Details)
		}
	}

	log.Info(
		"RunChecks merge completed",
		"results", len(results),
		"valid", valid,
		"agents_total", w.agentClient.AgentCount(),
		"agents_successful", len(responses),
		"shard_providers", len(assignment),
	)

	return results, assignment, nil
}

func (w *providersMasterWorker) fanoutRunChecks(
	ctx context.Context,
	base *providerchecksv1.RunChecksRequest,
	byEndpoint map[string][]*providerchecksv1.ProviderBatch,
) ([]agentrpc.RunChecksResult, []agentrpc.AgentCallError) {
	type workItem struct {
		endpoint  string
		providers []*providerchecksv1.ProviderBatch
	}
	items := make([]workItem, 0, len(byEndpoint))
	for endpoint, providers := range byEndpoint {
		if len(providers) == 0 {
			continue
		}
		items = append(items, workItem{endpoint: endpoint, providers: providers})
	}
	if len(items) == 0 {
		return nil, nil
	}

	results := make([]agentrpc.RunChecksResult, 0, len(items))
	errs := make([]agentrpc.AgentCallError, 0)
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, item := range items {
		wg.Add(1)
		go func(item workItem) {
			defer wg.Done()
			req := &providerchecksv1.RunChecksRequest{
				JobId:     base.GetJobId(),
				Providers: item.providers,
				Timeouts:  base.GetTimeouts(),
			}
			resp, err := w.agentClient.RunChecks(ctx, item.endpoint, req)
			if err != nil {
				mu.Lock()
				errs = append(errs, agentrpc.AgentCallError{Endpoint: item.endpoint, Err: err})
				mu.Unlock()
				return
			}
			mu.Lock()
			results = append(results, resp)
			mu.Unlock()
		}(item)
	}
	wg.Wait()
	return results, errs
}

func (w *providersMasterWorker) snapshotProviderAgentSuccess() map[string]string {
	w.providerAgentMu.Lock()
	defer w.providerAgentMu.Unlock()
	if len(w.providerAgentSuccess) == 0 {
		return nil
	}
	out := make(map[string]string, len(w.providerAgentSuccess))
	for k, v := range w.providerAgentSuccess {
		out[k] = v
	}
	return out
}

func (w *providersMasterWorker) updateProviderAgentSuccess(
	storageContracts []db.ContractToProviderRelation,
	results map[string]mergedContractCheck,
	assignment map[string]string,
) {
	if len(results) == 0 || len(assignment) == 0 {
		return
	}
	w.providerAgentMu.Lock()
	defer w.providerAgentMu.Unlock()
	if w.providerAgentSuccess == nil {
		w.providerAgentSuccess = make(map[string]string)
	}
	for _, sc := range storageContracts {
		row, ok := results[contractRelationKey(sc)]
		if !ok || row.Reason != constants.ValidStorageProof {
			continue
		}
		endpoint := assignment[sc.ProviderPublicKey]
		if endpoint == "" {
			continue
		}
		w.providerAgentSuccess[sc.ProviderPublicKey] = endpoint
	}
}

type mergedContractCheck struct {
	db.ContractProofsCheck
	Details string
	Stage   string
}

func contractRelationKey(sc db.ContractToProviderRelation) string {
	return sc.ProviderAddress + "|" + sc.Address
}

func (w *providersMasterWorker) buildRunChecksRPCRequest(jobID string, storageContracts []db.ContractToProviderRelation, availableProvidersIPs map[string]db.ProviderIP) (*providerchecksv1.RunChecksRequest, int, map[string]struct{}) {
	providersContracts := make(map[string][]db.ContractToProviderRelation)
	for _, sc := range storageContracts {
		providersContracts[sc.ProviderPublicKey] = append(providersContracts[sc.ProviderPublicKey], sc)
	}

	checkedKeys := make(map[string]struct{})
	providers := make([]*providerchecksv1.ProviderBatch, 0, len(providersContracts))
	for pubkey, contracts := range providersContracts {
		ip, ok := availableProvidersIPs[pubkey]
		if !ok || strings.TrimSpace(ip.Storage.IP) == "" || ip.Storage.Port <= 0 || len(ip.Storage.PublicKey) != ed25519.PublicKeySize {
			continue
		}
		contractRefs := make([]*providerchecksv1.ContractRef, 0, len(contracts))
		for _, c := range contracts {
			checkedKeys[contractRelationKey(c)] = struct{}{}
			contractRefs = append(contractRefs, &providerchecksv1.ContractRef{
				ContractAddress: c.Address,
				BagId:           c.BagID,
			})
		}
		providers = append(providers, &providerchecksv1.ProviderBatch{
			ProviderPubkey:  pubkey,
			ProviderAddress: contracts[0].ProviderAddress,
			StorageEndpoint: &providerchecksv1.Endpoint{
				Ip:         ip.Storage.IP,
				Port:       ip.Storage.Port,
				AdnlPubkey: append([]byte(nil), ip.Storage.PublicKey...),
			},
			Contracts: contractRefs,
		})
	}

	if strings.TrimSpace(jobID) == "" {
		jobID = fmt.Sprintf("storeproof-%d", time.Now().Unix())
	}

	return &providerchecksv1.RunChecksRequest{
		JobId:     jobID,
		Providers: providers,
		Timeouts: &providerchecksv1.CheckTimeouts{
			PingMs:  w.timeouts.PingMs,
			RldpMs:  w.timeouts.RldpMs,
			TotalMs: w.timeouts.TotalMs,
		},
	}, len(providers), checkedKeys
}

func mergeRunChecksResponses(storageContracts []db.ContractToProviderRelation, responses []agentrpc.RunChecksResult) (byKey map[string]mergedContractCheck, valid int) {
	type selectedResult struct {
		reason   constants.ReasonCode
		hasValue bool
		valid    bool
		details  string
		stage    string
	}

	merged := make(map[string]selectedResult, len(storageContracts))
	for _, agentResp := range responses {
		if agentResp.Response == nil {
			continue
		}
		for _, row := range agentResp.Response.GetResults() {
			if row == nil {
				continue
			}
			key := row.GetProviderAddress() + "|" + row.GetContractAddress()
			reasonCode := reasonFromProto(row.GetReasonCode())
			details := row.GetDetails()
			stage := pipelineevents.ParseStageFromDetails(details)

			current := merged[key]
			if !current.hasValue {
				merged[key] = selectedResult{
					reason:   reasonCode,
					hasValue: true,
					valid:    reasonCode == constants.ValidStorageProof,
					details:  details,
					stage:    stage,
				}
				continue
			}
			if current.valid {
				continue
			}
			if reasonCode == constants.ValidStorageProof {
				merged[key] = selectedResult{
					reason:   reasonCode,
					hasValue: true,
					valid:    true,
					details:  details,
					stage:    stage,
				}
			}
		}
	}

	valid = 0
	byKey = make(map[string]mergedContractCheck, len(storageContracts))
	for _, sc := range storageContracts {
		key := contractRelationKey(sc)
		chosen, ok := merged[key]
		reasonCode := constants.NotFound
		details := ""
		stage := pipelineevents.StageAgentRunChecksNoResult
		if ok && chosen.hasValue {
			reasonCode = chosen.reason
			details = chosen.details
			stage = chosen.stage
		}
		if reasonCode == constants.ValidStorageProof {
			valid++
		}
		byKey[key] = mergedContractCheck{
			ContractProofsCheck: db.ContractProofsCheck{
				ContractAddress: sc.Address,
				ProviderAddress: sc.ProviderAddress,
				Reason:          reasonCode,
			},
			Details: details,
			Stage:   stage,
		}
	}

	return byKey, valid
}

func mergeStorageRatesResponses(pubkeys []string, responses []agentrpc.RunStorageRatesResult) map[string]*providerchecksv1.StorageRatesResult {
	type selectedRates struct {
		row   *providerchecksv1.StorageRatesResult
		valid bool
	}
	byPubkey := make(map[string]selectedRates, len(pubkeys))
	for _, resp := range responses {
		if resp.Response == nil {
			continue
		}
		for _, row := range resp.Response.GetResults() {
			if row == nil {
				continue
			}
			key := strings.TrimSpace(row.GetProviderPubkey())
			if key == "" {
				continue
			}

			current, ok := byPubkey[key]
			if !ok {
				byPubkey[key] = selectedRates{row: row, valid: row.GetOk()}
				continue
			}
			if current.valid {
				continue
			}
			if row.GetOk() {
				byPubkey[key] = selectedRates{row: row, valid: true}
			}
		}
	}

	out := make(map[string]*providerchecksv1.StorageRatesResult, len(pubkeys))
	for _, pk := range pubkeys {
		if selected, ok := byPubkey[pk]; ok {
			out[pk] = selected.row
		}
	}
	return out
}

func checkProviderFiles(ctx context.Context, gw *adnl.Gateway, ip db.ProviderIP, storageContracts []db.ContractToProviderRelation, bagsStatuses *sync.Map, log *slog.Logger) {
	log = log.With(slog.String("provider_pubkey", ip.PublicKey))
	log.Debug("Start checking provider files")
	s := time.Now()
	defer func() {
		log.Debug("Finished checking provider files", "duration", time.Since(s).String())
	}()

	stats := make(map[constants.ReasonCode]int)
	// to skip dead providers and save time
	maxFailureThreshold := uint32(float32(len(storageContracts)) / 100.0 * 20.0)
	var failsInARow uint32

	addr := ip.Storage.IP + ":" + strconv.Itoa(int(ip.Storage.Port))
	peer, rErr := gw.RegisterClient(addr, ip.Storage.PublicKey)
	if rErr != nil {
		log.Debug("failed to create ADNL peer", "error", rErr)
		fillStatuses(bagsStatuses, storageContracts, constants.CantCreatePeer)
		return
	}

	pingCtx, pingCancel := context.WithTimeout(ctx, pingTimeout)
	_, pErr := peer.Ping(pingCtx)
	pingCancel()
	if pErr != nil {
		log.Debug("initial provider ping failed", "error", pErr)
		fillStatuses(bagsStatuses, storageContracts, constants.FailedInitialPing)
		return
	}

	rl := rldp.NewClientV2(peer)
	defer rl.Close()

	for _, sc := range storageContracts {
		statusKey := getKey(sc.BagID, ip.Storage.IP, ip.Storage.Port)

		if failsInARow > maxFailureThreshold {
			bagsStatuses.Store(statusKey, db.ContractProofsCheck{
				ContractAddress: sc.Address,
				ProviderAddress: sc.ProviderAddress,
				Reason:          constants.UnavailableProvider,
			})
			log.Info("skip", "bag_id", sc.BagID)
			continue
		}

		reason := checkPiece(ctx, rl, sc.BagID, log)
		bagsStatuses.Store(statusKey, db.ContractProofsCheck{
			ContractAddress: sc.Address,
			ProviderAddress: sc.ProviderAddress,
			Reason:          reason,
		})

		stats[reason]++

		if reason == constants.ValidStorageProof {
			failsInARow = 0
		} else {
			failsInARow++
		}

		// weak providers may be overloaded
		time.Sleep(500 * time.Millisecond)
	}

	for reason, count := range stats {
		log.Debug("checked provider files", "reason", int(reason), "count", count)
	}
}

func checkPiece(ctx context.Context, rl *rldp.RLDP, bagID string, log *slog.Logger) (reason constants.ReasonCode) {
	log = log.With(slog.String("bag_id", bagID))

	reason = constants.NotFound

	peer, ok := rl.GetADNL().(adnl.Peer)
	if !ok {
		log.Error("failed to get ADNL peer")
		reason = constants.UnknownPeer
		return
	}

	// in case connection was lost
	peer.Reinit()
	// peer can be closed after some time, so for extra stability we reinit before each operation if needed
	est := time.Now()

	pingCtx, pc := context.WithTimeout(ctx, pingTimeout)
	_, err := peer.Ping(pingCtx)
	pc()
	if err != nil {
		log.Debug("ping to provider failed", "error", err)
		reason = constants.PingFailed
		return
	}

	bag, dErr := hex.DecodeString(bagID)
	if dErr != nil {
		log.Error("failed to decode bag ID", "error", dErr)
		reason = constants.InvalidBagID
		return
	}

	over, err := tl.Hash(keys.PublicKeyOverlay{Key: bag})
	if err != nil {
		log.Debug("failed to hash overlay key", "error", err)
		reason = constants.InvalidBagID
		return
	}

	if time.Since(est) > 5*time.Second {
		peer.Reinit()
		est = time.Now()
	}

	// get torrent info
	var res tonstorage.TorrentInfoContainer
	rlCtx, rlc := context.WithTimeout(ctx, rlQueryTimeout)
	err = rl.DoQuery(rlCtx, 32<<20, overlay.WrapQuery(over, &tonstorage.GetTorrentInfo{}), &res)
	rlc()
	if err != nil {
		log.Debug("failed to get torrent info from provider", "error", err)
		reason = constants.GetInfoFailed
		return
	}

	cl, err := cell.FromBOC(res.Data)
	if err != nil {
		log.Debug("failed to parse BoC of torrent info", "error", err)
		reason = constants.InvalidHeader
		return
	}

	if !bytes.Equal(cl.Hash(), bag) {
		log.Debug("hash not equal bag", "hash", cl.Hash(), "bag", bag)
		reason = constants.InvalidHeader
		return
	}

	var info tonstorage.TorrentInfo
	err = tlb.Parse(&info, cl)
	if err != nil {
		log.Debug("failed to load torrent info from cell", "error", err)
		reason = constants.InvalidHeader
		return
	}

	pieceID := int32(1)
	var p int32
	if info.PieceSize != 0 {
		p = int32(info.FileSize / uint64(info.PieceSize))
	}
	if p != 0 {
		pieceID = rand.Int31n(p)
	}

	if time.Since(est) > 5*time.Second {
		peer.Reinit()
	}

	// get piece proof and validate
	var piece tonstorage.Piece
	rl2Ctx, rl2c := context.WithTimeout(ctx, rlQueryTimeout)
	err = rl.DoQuery(rl2Ctx, 32<<20, overlay.WrapQuery(over, &tonstorage.GetPiece{PieceID: pieceID}), &piece)
	rl2c()

	if err != nil {
		log.Debug("failed to get piece from provider", "error", err)
		reason = constants.CantGetPiece
		return
	}

	proof, err := cell.FromBOC(piece.Proof)
	if err != nil {
		log.Debug("failed to parse BoC of piece", "error", err)
		reason = constants.CantParseBoC
		return
	}

	err = cell.CheckProof(proof, info.RootHash)
	if err != nil {
		log.Debug("proof check failed", "error", err)
		reason = constants.ProofCheckFailed
		return
	}

	reason = constants.ValidStorageProof
	return
}

// updateRejectedContracts check contracts balance and providers list to mark contracts as rejected
// returns list of active contracts
func (w *providersMasterWorker) updateRejectedContracts(ctx context.Context, storageContracts []db.ContractToProviderRelation) (activeContracts []db.ContractToProviderRelation, err error) {
	log := w.logger.With(slog.String("worker", "updateRejectedContracts"))

	if len(storageContracts) == 0 {
		log.Debug("no storage contracts to process")
		return
	}

	uniqueContractAddresses := make(map[string]uint64, len(storageContracts))
	for _, sc := range storageContracts {
		uniqueContractAddresses[sc.Address] = sc.Size
	}

	contractAddresses := make([]string, 0, len(uniqueContractAddresses))
	for addr := range uniqueContractAddresses {
		contractAddresses = append(contractAddresses, addr)
	}

	contractsProvidersList, err := w.ton.GetProvidersInfo(ctx, contractAddresses)
	if err != nil {
		log.Error("failed to get providers info", "error", err)
		return
	}

	type contractInfo struct {
		providers map[string]struct{}
		skip      bool
	}

	// map of storage contract addresses to their active providers
	activeRelations := make(map[string]contractInfo, len(contractsProvidersList))
	for _, contract := range contractsProvidersList {
		contractProviders := make(map[string]struct{}, len(contract.Providers))
		for _, provider := range contract.Providers {
			providerPublicKey := fmt.Sprintf("%x", provider.Key)
			if isRemovedByLowBalance(new(big.Int).SetUint64(uniqueContractAddresses[contract.Address]), provider, contract) {
				log.Warn("storage contract has not enough balance for too long, will be removed",
					"provider", providerPublicKey,
					"address", contract.Address,
					"balance", contract.Balance)
				continue
			}

			contractProviders[providerPublicKey] = struct{}{}
		}

		// in case no available lite servers use skip, to not remove contracts from db
		activeRelations[contract.Address] = contractInfo{
			providers: contractProviders,
			skip:      contract.LiteServerError,
		}
	}

	activeContracts = make([]db.ContractToProviderRelation, 0, len(storageContracts))
	closedContracts := make([]db.ContractToProviderRelation, 0, len(storageContracts))

	for _, sc := range storageContracts {
		if contractInfo, exists := activeRelations[sc.Address]; exists {
			if contractInfo.skip {
				log.Debug("lite servers is not available, keep contract active for current cycle", "address", sc.Address)
				activeContracts = append(activeContracts, sc)
				continue
			}

			if _, providerExists := contractInfo.providers[sc.ProviderPublicKey]; providerExists {
				activeContracts = append(activeContracts, sc)
			} else {
				closedContracts = append(closedContracts, sc)
			}
		} else {
			closedContracts = append(closedContracts, sc)
		}
	}

	err = w.providers.UpdateRejectedStorageContracts(ctx, closedContracts)
	if err != nil {
		log.Error("failed to update rejected storage contracts", "error", err)
		return nil, err
	}

	log.Info("successfully updated rejected storage contracts",
		"closed_count", len(closedContracts),
		"active_count", len(activeContracts))

	return
}

func (w *providersMasterWorker) updateProvidersIPs(ctx context.Context, storageContracts []db.ContractToProviderRelation, recorder *pipelineevents.Recorder) (availableProvidersIPs map[string]db.ProviderIP, err error) {
	return w.refreshProvidersEndpoints(ctx, storageContracts, false, recorder)
}

func (w *providersMasterWorker) findStorageIPOverlay(ctx context.Context, providerIP string, contracts []db.ContractToProviderRelation, known db.IPInfo, log *slog.Logger) (ip db.IPInfo, err error) {
	if len(contracts) == 0 {
		err = fmt.Errorf("no contracts provided")
		return
	}

	bagsToCheck := len(contracts)
	switch {
	case len(contracts) > 100:
		bagsToCheck = max(1, len(contracts)*10/100)
	case len(contracts) > 5:
		bagsToCheck = max(1, len(contracts)*20/100)
	}

	log = log.With("provider_ip", providerIP, "bags_to_check", bagsToCheck, "total_bags", len(contracts))

	shuffled := make([]db.ContractToProviderRelation, len(contracts))
	copy(shuffled, contracts)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	candidates := make(map[string]*overlayStorageCandidate)

	for i := 0; i < bagsToCheck && i < len(shuffled); i++ {
		sc := shuffled[i]

		bag, dErr := hex.DecodeString(sc.BagID)
		if dErr != nil {
			log.Error("failed to decode bag ID", "bag_id", sc.BagID, "error", dErr)
			continue
		}

		dhtTimeoutCtx, cancel := context.WithTimeout(ctx, dhtTimeout)
		nodesList, _, fErr := w.dhtClient.FindOverlayNodes(dhtTimeoutCtx, bag)
		cancel()

		if fErr != nil {
			if !errors.Is(fErr, dht.ErrDHTValueIsNotFound) {
				log.Error("failed to find bag overlay nodes", "bag_id", sc.BagID, "error", fErr)
			}
			continue
		}

		if nodesList == nil || len(nodesList.List) == 0 {
			log.Debug("no peers found for bag in DHT", "bag_id", sc.BagID)
			continue
		}

		seen := make(map[string]struct{})
		for _, node := range nodesList.List {
			key, ok := node.ID.(keys.PublicKeyED25519)
			if !ok {
				continue
			}

			adnlID, hErr := tl.Hash(key)
			if hErr != nil {
				log.Error("failed to hash overlay key", "error", hErr)
				continue
			}

			dhtTimeoutCtx2, cancel2 := context.WithTimeout(ctx, dhtTimeout)
			addrList, pubKey, fErr := w.dhtClient.FindAddresses(dhtTimeoutCtx2, adnlID)
			cancel2()

			if fErr != nil {
				if !errors.Is(fErr, dht.ErrDHTValueIsNotFound) {
					log.Debug("failed to find addresses in DHT", "error", fErr)
				}
				continue
			}

			if addrList == nil || len(addrList.Addresses) == 0 {
				continue
			}

			for _, addr := range addrList.Addresses {
				addrIP := adnladdress.IPValue(addr)
				if addrIP == nil {
					continue
				}
				if addrIP.String() != providerIP {
					continue
				}

				id := overlayCandidateKey(pubKey)
				if _, ok := seen[id]; ok {
					continue
				}
				seen[id] = struct{}{}

				c := candidates[id]
				if c == nil {
					c = &overlayStorageCandidate{
						PublicKey: append(ed25519.PublicKey(nil), pubKey...),
						IP:        addrIP.String(),
						Port:      adnladdress.PortValue(addr),
					}
					candidates[id] = c
				}
				c.Bags++
				break
			}
		}
	}

	ip, err = pickOverlayStorage(candidates, known)
	if err != nil {
		if len(candidates) == 0 {
			err = fmt.Errorf("storage IP not found via overlay DHT after checking %d bags", bagsToCheck)
		}
		return
	}

	log.Info("found storage IP via overlay DHT",
		"candidates", len(candidates),
		"top_bags", overlayTopBags(candidates),
		"known_bags", overlayCandidateBags(candidates, known.PublicKey),
		"ip", ip.IP,
		"port", ip.Port)
	return
}

func (w *providersMasterWorker) findProviderIPs(ctx context.Context, sc db.ContractToProviderRelation, log *slog.Logger) (result db.ProviderIP, err error) {
	log = log.With("provider_pubkey", sc.ProviderPublicKey)

	result.PublicKey = sc.ProviderPublicKey

	addr, err := address.ParseAddr(sc.Address)
	if err != nil {
		log.Error("failed to parse address", "address", sc.Address, "error", err)
		return
	}

	pk, err := hex.DecodeString(sc.ProviderPublicKey)
	if err != nil {
		log.Error("failed to decode provider public key", "error", err)
		return
	}

	result.Provider, err = w.findProviderIP(ctx, pk)
	if err != nil {
		log.Error("failed to verify provider IP", "error", err)
		return
	}

	result.Storage, err = w.findStorageIP(ctx, addr, pk)
	if err != nil {
		log.Error("failed to find storage IP", "address", sc.Address, "error", err)
		return
	}

	return
}

func (w *providersMasterWorker) findStorageIP(ctx context.Context, addr *address.Address, pk []byte) (ip db.IPInfo, err error) {
	var proof []byte
	err = utils.TryNTimes(func() (cErr error) {
		timeoutCtx, cancel := context.WithTimeout(ctx, providerResponseTimeout)
		defer cancel()

		proof, cErr = w.providerClient.VerifyStorageADNLProof(timeoutCtx, pk, addr)
		return
	}, verifyStorageRetries)
	if err != nil {
		err = stageError(pipelineevents.StageVerifyStorageADNLProof, fmt.Errorf("failed to verify storage adnl proof: %w", err))
		return
	}

	dhtTimeoutCtx, cancel := context.WithTimeout(ctx, dhtTimeout)
	defer cancel()
	l, pub, err := w.dhtClient.FindAddresses(dhtTimeoutCtx, proof)
	if err != nil {
		err = stageError(pipelineevents.StageDHTFindStorageAddresses, fmt.Errorf("failed to find addresses in dht: %w", err))
		return
	}

	if l == nil || len(l.Addresses) == 0 {
		err = stageError(pipelineevents.StageDHTFindStorageAddresses, fmt.Errorf("no storage addresses found"))
		return
	}

	ip.PublicKey = pub
	ipVal := adnladdress.IPValue(l.Addresses[0])
	if ipVal == nil {
		err = stageError(pipelineevents.StageDHTFindStorageAddresses, fmt.Errorf("empty storage address ip"))
		return
	}
	ip.IP = ipVal.String()
	ip.Port = adnladdress.PortValue(l.Addresses[0])

	return
}

func (w *providersMasterWorker) findProviderIP(ctx context.Context, pk []byte) (ip db.IPInfo, err error) {
	channelKeyId, err := tl.Hash(keys.PublicKeyED25519{Key: pk})
	if err != nil {
		err = stageError(pipelineevents.StageDHTFindProviderRecord, fmt.Errorf("failed to calc hash of provider key: %w", err))
		return
	}

	dhtTimeoutCtx, cancel := context.WithTimeout(ctx, dhtTimeout)
	defer cancel()
	dhtVal, _, err := w.dhtClient.FindValue(dhtTimeoutCtx, &dht.Key{
		ID:    channelKeyId,
		Name:  []byte("storage-provider"),
		Index: 0,
	})
	if err != nil {
		err = stageError(pipelineevents.StageDHTFindProviderRecord, fmt.Errorf("failed to find storage-provider in dht: %w", err))
		return
	}

	var nodeAddr transport.ProviderDHTRecord
	if _, pErr := tl.Parse(&nodeAddr, dhtVal.Data, true); pErr != nil {
		err = stageError(pipelineevents.StageDHTFindProviderRecord, fmt.Errorf("failed to parse node dht value: %w", pErr))
		return
	}

	if len(nodeAddr.ADNLAddr) == 0 {
		err = stageError(pipelineevents.StageDHTFindProviderRecord, fmt.Errorf("no adnl addresses in node dht value"))
		return
	}

	dhtTimeoutCtx2, cancel2 := context.WithTimeout(ctx, dhtTimeout)
	defer cancel2()
	l, pub, fErr := w.dhtClient.FindAddresses(dhtTimeoutCtx2, nodeAddr.ADNLAddr)
	if fErr != nil {
		err = stageError(pipelineevents.StageDHTFindProviderAddresses, fmt.Errorf("failed to find adnl addresses in dht: %w", fErr))
		return
	}

	if l == nil || len(l.Addresses) == 0 {
		err = stageError(pipelineevents.StageDHTFindProviderAddresses, fmt.Errorf("no provider addresses found"))
		return
	}

	ip.PublicKey = pub
	ipVal := adnladdress.IPValue(l.Addresses[0])
	if ipVal == nil {
		err = stageError(pipelineevents.StageDHTFindProviderAddresses, fmt.Errorf("empty provider address ip"))
		return
	}
	ip.IP = ipVal.String()
	ip.Port = adnladdress.PortValue(l.Addresses[0])

	return
}

func (w *providersMasterWorker) scanProviderTransactions(ctx context.Context, provider db.ProviderWallet) (contracts map[string]db.StorageContract, lastLT uint64, err error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, getTxTimeout)
	defer cancel()

	txs, err := w.ton.GetTransactions(timeoutCtx, provider.Address, provider.LT)
	if err != nil {
		err = fmt.Errorf("failed to get transactions error: %w", err)
		return
	}

	contracts = make(map[string]db.StorageContract, len(txs))

	lastLT = provider.LT
	for _, tx := range txs {
		if tx == nil {
			continue
		}

		if tx.Op != storageRewardWithdrawalOpCode {
			continue
		}

		s := db.StorageContract{
			ProvidersAddresses: make(map[string]struct{}),
			Address:            tx.From,
			LastLT:             tx.LT,
		}
		s.ProvidersAddresses[provider.Address] = struct{}{}

		if tx.LT > lastLT {
			lastLT = tx.LT
		}

		contracts[tx.From] = s
	}

	return
}

func NewWorker(
	providers providers,
	system system,
	ton ton,
	providerClient *transport.Client,
	dhtClient *dht.Client,
	ipinfo ipclient,
	agentClient agentclient,
	masterAddr string,
	batchSize uint32,
	timeouts RunChecksTimeouts,
	endpointCfg EndpointRefreshConfig,
	storeProofCfg StoreProofConfig,
	logger *slog.Logger,
) Worker {
	_, prv, err := ed25519.GenerateKey(nil)
	if err != nil {
		logger.Error("failed to generate ed25519 key", "error", err)
		return nil
	}

	if endpointCfg.StaleTTL <= 0 {
		endpointCfg.StaleTTL = defaultEndpointStaleTTL
	}
	if endpointCfg.FullRefreshInterval <= 0 {
		endpointCfg.FullRefreshInterval = defaultEndpointFullRefreshInterval
	}
	if endpointCfg.FailInterval <= 0 {
		endpointCfg.FailInterval = defaultEndpointFullRefreshFailInterval
	}
	if storeProofCfg.Interval <= 0 {
		storeProofCfg.Interval = defaultStoreProofInterval
	}

	return &providersMasterWorker{
		providers:            providers,
		system:               system,
		ton:                  ton,
		prv:                  prv,
		providerClient:       providerClient,
		agentClient:          agentClient,
		dhtClient:            dhtClient,
		ipinfo:               ipinfo,
		masterAddr:           masterAddr,
		batchSize:            batchSize,
		timeouts:             timeouts,
		endpointCfg:          endpointCfg,
		storeProofCfg:        storeProofCfg,
		logger:               logger,
		lastKnownIPs:         make(map[string]db.ProviderIP),
		providerAgentSuccess: make(map[string]string),
	}
}
