package providersmaster

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/adnl/dht"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-storage-provider/pkg/transport"

	"mytonprovider-backend/pkg/clients/ifconfig"
	tonclient "mytonprovider-backend/pkg/clients/ton"
	"mytonprovider-backend/pkg/models/db"
	"mytonprovider-backend/pkg/utils"
)

const (
	validProvider               = 0
	invalidAddress              = 1
	invalidProviderPublicKey    = 2
	invalidStorageProofResponse = 3
	verifyStorageRetries        = 3
	invalidSize                 = 4
	unavailableProvider         = 500
)

const (
	lastLTKey                     = "masterWalletLastLT"
	prefix                        = "tsp-"
	storageRewardWithdrawalOpCode = 0xa91baf56
	maxConcurrentProviderChecks   = 10
	fakeSize                      = 1
	providerResponseTimeout       = 5 * time.Second
	providerCheckTimeout          = 10 * time.Second
	ipInfoTimeout                 = 10 * time.Second
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
	UpdateProvidersIPInfo(ctx context.Context, ips []db.ProviderIPInfo) (err error)
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

type providersMasterWorker struct {
	providers      providers
	system         system
	ton            ton
	ipinfo         ipclient
	providerClient *transport.Client
	dhtClient      *dht.Client
	masterAddr     string
	batchSize      uint32
	logger         *slog.Logger
}

type Worker interface {
	CollectNewProviders(ctx context.Context) (interval time.Duration, err error)
	UpdateKnownProviders(ctx context.Context) (interval time.Duration, err error)
	CollectProvidersNewStorageContracts(ctx context.Context) (interval time.Duration, err error)
	StoreProof(ctx context.Context) (interval time.Duration, err error)
	UpdateUptime(ctx context.Context) (interval time.Duration, err error)
	UpdateRating(ctx context.Context) (interval time.Duration, err error)
	UpdateIPInfo(ctx context.Context) (interval time.Duration, err error)
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

	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
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

	providersInfo := make([]db.ProviderUpdate, 0, len(p))
	providersStatuses := make([]db.ProviderStatusUpdate, 0, len(p))
	for _, pubkey := range p {
		select {
		case <-ctx.Done():
			log.Info("context done, stopping provider check")
			return
		default:
		}
		d, err := hex.DecodeString(pubkey)
		if err != nil || len(d) != 32 {
			continue
		}

		timeoutCtx, cancel := context.WithTimeout(ctx, providerResponseTimeout)
		rates, err := w.providerClient.GetStorageRates(timeoutCtx, d, fakeSize)
		cancel()
		if err != nil {
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
			RatePerMBDay: new(big.Int).SetBytes(rates.RatePerMBDay).Int64(),
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
	const (
		successInterval = 60 * time.Minute
		failureInterval = 15 * time.Second
	)

	log := w.logger.With(slog.String("worker", "StoreProof"))
	log.Debug("checking storage proofs")

	interval = successInterval

	storageContracts, err := w.providers.GetStorageContracts(ctx)
	if err != nil {
		log.Error("failed to get storage contracts", "error", err)
		interval = failureInterval

		return
	}

	storageContracts, err = w.updateRejectedContracts(ctx, storageContracts)
	if err != nil {
		interval = failureInterval
		return
	}

	availableProvidersIPs, err := w.updateProvidersIPs(ctx, storageContracts)
	if err != nil {
		interval = failureInterval
		return
	}

	err = w.updateActiveContracts(ctx, storageContracts, availableProvidersIPs)
	if err != nil {
		interval = failureInterval
		return
	}

	err = w.providers.UpdateStatuses(ctx)
	if err != nil {
		log.Error("failed to update provider statuses", "error", err)
		interval = failureInterval
		return
	}

	return
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
		successInterval = 120 * time.Minute
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
		time.Sleep(1 * time.Second)

		ipErr := func() error {
			timeoutCtx, cancel := context.WithTimeout(ctx, ipInfoTimeout)
			defer cancel()

			info, err := w.ipinfo.GetIPInfo(timeoutCtx, ip.IP)
			if err != nil {
				return fmt.Errorf("failed to get IP info: %w", err)
			}

			s, err := json.Marshal(info)
			if err != nil {
				return fmt.Errorf("failed to marshal IP info: %w, ip: %s, info: %s", err, ip.IP, info)
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

func (w *providersMasterWorker) updateActiveContracts(ctx context.Context, storageContracts []db.ContractToProviderRelation, availableProvidersIPs map[string]db.ProviderIP) (err error) {
	log := w.logger.With(slog.String("worker", "StoreProof"), slog.String("function", "updateActiveContracts"))

	if len(storageContracts) == 0 || len(availableProvidersIPs) == 0 {
		return nil
	}

	semaphore := make(chan struct{}, maxConcurrentProviderChecks)
	resultChan := make(chan db.ContractProofsCheck, len(storageContracts))

	var wg sync.WaitGroup

	for _, sc := range storageContracts {
		wg.Add(1)
		go func(contract db.ContractToProviderRelation) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			proofCheck := w.processContractProof(ctx, contract, availableProvidersIPs, log)
			resultChan <- proofCheck
		}(sc)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	contractProofsChecks := make([]db.ContractProofsCheck, 0, len(storageContracts))
	for proofCheck := range resultChan {
		contractProofsChecks = append(contractProofsChecks, proofCheck)
	}

	err = w.providers.UpdateContractProofsChecks(ctx, contractProofsChecks)
	if err != nil {
		log.Error("failed to update contract proofs checks", "error", err)
		return
	}

	log.Info("successfully updated contract proofs checks", "count", len(contractProofsChecks))
	return nil
}

func (w *providersMasterWorker) processContractProof(ctx context.Context, sc db.ContractToProviderRelation, availableProvidersIPs map[string]db.ProviderIP, log *slog.Logger) (result db.ContractProofsCheck) {
	timestamp := time.Now()

	result = db.ContractProofsCheck{
		Address:         sc.Address,
		ProviderAddress: sc.ProviderAddress,
		Timestamp:       timestamp,
		Reason:          unavailableProvider,
	}

	if providerIP, exists := availableProvidersIPs[sc.ProviderPublicKey]; exists && providerIP.IP == "" {
		return result
	}

	if reason, ok := w.validateContractData(sc); !ok {
		log.Error("invalid contract data", "address", sc.Address, "provider", sc.ProviderPublicKey, "reason", reason)
		result.Reason = reason
		return result
	}

	proofResult := w.verifyStorageProof(ctx, sc, log)
	result.Reason = proofResult

	return result
}

func (w *providersMasterWorker) validateContractData(sc db.ContractToProviderRelation) (reason uint32, ok bool) {
	if sc.Address == "" {
		reason = invalidAddress
		return
	}

	if sc.ProviderPublicKey == "" {
		reason = invalidProviderPublicKey
		return
	}

	if sc.Size <= 0 {
		reason = invalidSize
		return
	}

	ok = true

	return
}

func (w *providersMasterWorker) verifyStorageProof(ctx context.Context, sc db.ContractToProviderRelation, log *slog.Logger) uint32 {
	addr, err := address.ParseAddr(sc.Address)
	if err != nil {
		log.Error("failed to parse contract address", "address", sc.Address, "error", err)
		return invalidAddress
	}

	providerKey, err := hex.DecodeString(sc.ProviderPublicKey)
	if err != nil {
		log.Error("failed to decode provider public key", "provider", sc.ProviderPublicKey, "error", err)
		return invalidProviderPublicKey
	}

	randomBig, err := rand.Int(rand.Reader, big.NewInt(int64(sc.Size)))
	if err != nil {
		log.Error("failed to generate random number", "provider", sc.ProviderPublicKey, "address", sc.Address, "error", err)
		randomBig = big.NewInt(0)
	}

	toProof := randomBig.Uint64()

	requestCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	stResp, err := w.providerClient.RequestStorageInfo(requestCtx, providerKey, addr, toProof)
	if err != nil {
		log.Error("failed to request storage info", "provider", sc.ProviderPublicKey, "address", sc.Address, "error", err)
		return unavailableProvider
	}

	if stResp == nil || len(stResp.Proof) == 0 {
		log.Error("invalid storage proof response", "provider", sc.ProviderPublicKey, "address", sc.Address, "response", stResp)
		return invalidStorageProofResponse
	}

	return validProvider
}

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

	// map of storage contract addresses to their active providers
	activeRelations := make(map[string]map[string]struct{}, len(contractsProvidersList))
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

		if len(contractProviders) != 0 {
			activeRelations[contract.Address] = contractProviders
		}
	}

	activeContracts = make([]db.ContractToProviderRelation, 0, len(storageContracts))
	closedContracts := make([]db.ContractToProviderRelation, 0, len(storageContracts))

	for _, sc := range storageContracts {
		if contractProviders, exists := activeRelations[sc.Address]; exists {
			if _, providerExists := contractProviders[sc.ProviderPublicKey]; providerExists {
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

func isRemovedByLowBalance(bagSize *big.Int, provider tonclient.Provider, contract tonclient.StorageContractProviders) bool {
	var storageFee = tlb.MustFromTON("0.05").Nano()

	mul := new(big.Int).Mul(new(big.Int).SetUint64(provider.RatePerMBDay), bagSize)
	mul = mul.Mul(mul, new(big.Int).SetUint64(uint64(provider.MaxSpan)))
	bounty := new(big.Int).Div(mul, big.NewInt(24*60*60*1024*1024))
	bounty = bounty.Add(bounty, storageFee)

	if new(big.Int).SetUint64(contract.Balance).Cmp(bounty) < 0 {
		var deadline int64
		fresh := provider.LastProofTime.Unix() <= 0
		if fresh {
			return false
		} else {
			deadline = provider.LastProofTime.Unix() + int64(provider.MaxSpan) + 3600
		}

		if time.Now().Unix() > deadline {
			return true
		}
	}

	return false
}

func (w *providersMasterWorker) updateProvidersIPs(ctx context.Context, storageContracts []db.ContractToProviderRelation) (availableProvidersIPs map[string]db.ProviderIP, err error) {
	log := w.logger.With(slog.String("worker", "StoreProof"), slog.String("function", "updateProvidersIPs"))

	if len(storageContracts) == 0 {
		log.Debug("no storage contracts to process for IP update")
		return nil, nil
	}

	availableProvidersIPs = make(map[string]db.ProviderIP, len(storageContracts))

	uniqueProviders := make(map[string]db.ContractToProviderRelation)
	for _, sc := range storageContracts {
		if _, exists := uniqueProviders[sc.ProviderPublicKey]; !exists {
			uniqueProviders[sc.ProviderPublicKey] = sc
		}
	}

	semaphore := make(chan struct{}, maxConcurrentProviderChecks)

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, sc := range uniqueProviders {
		wg.Add(1)
		go func(contract db.ContractToProviderRelation) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			providerIP := w.processProviderIP(ctx, contract, log)

			mu.Lock()
			availableProvidersIPs[contract.ProviderPublicKey] = providerIP
			mu.Unlock()
		}(sc)
	}

	wg.Wait()

	ips := make([]db.ProviderIP, 0, len(availableProvidersIPs))
	for _, p := range availableProvidersIPs {
		ips = append(ips, p)
	}

	err = w.providers.UpdateProvidersIPs(ctx, ips)
	if err != nil {
		log.Error("failed to update providers IPs", "error", err)
		return
	}

	log.Info("successfully updated providers IPs", "count", len(availableProvidersIPs))
	return
}

func (w *providersMasterWorker) processProviderIP(ctx context.Context, sc db.ContractToProviderRelation, log *slog.Logger) (result db.ProviderIP) {
	result.PublicKey = sc.ProviderPublicKey

	addr, err := address.ParseAddr(sc.Address)
	if err != nil {
		log.Error("failed to parse address", "address", sc.Address, "error", err)
		return result
	}

	pk, err := hex.DecodeString(sc.ProviderPublicKey)
	if err != nil {
		log.Error("failed to decode provider public key", "provider", sc.ProviderPublicKey, "error", err)
		return result
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, providerCheckTimeout)
	defer cancel()

	var proof []byte
	err = utils.TryNTimes(func() (cErr error) {
		proof, cErr = w.providerClient.VerifyStorageADNLProof(timeoutCtx, pk, addr)
		return
	}, verifyStorageRetries)
	if err != nil {
		log.Error("failed to verify storage adnl proof", "provider", sc.ProviderPublicKey, "error", err)
		return result
	}

	l, _, err := w.dhtClient.FindAddresses(ctx, proof)
	if err != nil {
		log.Error("failed to find provider addresses", "provider", sc.ProviderPublicKey, "error", err)
		return result
	}

	if l == nil || len(l.Addresses) == 0 {
		log.Warn("no addresses found for provider", "provider", sc.ProviderPublicKey)
		return result
	}

	return db.ProviderIP{
		PublicKey: sc.ProviderPublicKey,
		IP:        l.Addresses[0].IP.String(),
		Port:      l.Addresses[0].Port,
	}
}

func (w *providersMasterWorker) scanProviderTransactions(ctx context.Context, provider db.ProviderWallet) (contracts map[string]db.StorageContract, lastLT uint64, err error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
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
	masterAddr string,
	batchSize uint32,
	logger *slog.Logger,
) Worker {
	return &providersMasterWorker{
		providers:      providers,
		system:         system,
		ton:            ton,
		providerClient: providerClient,
		dhtClient:      dhtClient,
		ipinfo:         ipinfo,
		masterAddr:     masterAddr,
		batchSize:      batchSize,
		logger:         logger,
	}
}
