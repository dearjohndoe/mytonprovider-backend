package providersmaster

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/adnl/dht"
	"github.com/xssnick/tonutils-storage-provider/pkg/transport"

	"mytonprovider-backend/pkg/models/db"
	"mytonprovider-backend/pkg/tonclient"
)

const (
	lastLTKey                     = "masterWalletLastLT"
	prefix                        = "tsp-"
	storageRewardWithdrawalOpCode = 0xa91baf56
	fakeSize                      = 1
	providerResponseTimeout       = 5 * time.Second
)

type providers interface {
	GetAllProvidersPubkeys(ctx context.Context) (pubkeys []string, err error)
	GetAllProvidersWallets(ctx context.Context) (wallets []db.ProviderWallet, err error)
	UpdateProvidersLT(ctx context.Context, providers []db.ProviderWalletLT) (err error)
	AddStorageContracts(ctx context.Context, contracts []db.StorageContract) (err error)
	GetStorageContracts(ctx context.Context) (contracts []db.StorageContractShort, err error)
	AddProviders(ctx context.Context, providers []db.ProviderCreate) (err error)
	UpdateProvidersIPs(ctx context.Context, ips []db.ProviderIP) (err error)
	UpdateProviders(ctx context.Context, providers []db.ProviderUpdate) (err error)
	AddStatuses(ctx context.Context, providers []db.ProviderStatusUpdate) (err error)
	UpdateContractProofsChecks(ctx context.Context, contractsProofs []db.ContractProofsCheck) (err error)
	UpdateUptime(ctx context.Context) (err error)
	UpdateRating(ctx context.Context) (err error)
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

type providersMasterWorker struct {
	providers      providers
	system         system
	ton            ton
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
		go func(ctx context.Context) {
			defer wg.Done()

			// TODO: retries on errors
			var lastLT uint64
			sc, lastLT, err := w.scanProviderTransactions(ctx, provider)
			if err != nil {
				log.Error("failed to scan provider transactions", "address", provider.Address, "error", err)
			}

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

			if lastLT != provider.LT {
				pmu.Lock()
				providersToUpdate = append(providersToUpdate, db.ProviderWalletLT{
					PubKey: provider.PubKey,
					LT:     lastLT,
				})
				pmu.Unlock()
			}
		}(ctx)
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

		if contract.Size == 0 || contract.ChunkSize == 0 {
			log.Error("invalid storage contract size or chunk size", "address", contract.Address, "size", contract.Size, "chunkSize", contract.ChunkSize)
			continue
		}

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

	providersIPs := make(map[string]db.ProviderIP, len(storageContracts))
	for _, sc := range storageContracts {
		if _, ok := providersIPs[sc.ProviderPublicKey]; ok {
			continue
		}

		addr, aErr := address.ParseAddr(sc.Address)
		if aErr != nil {
			log.Error("failed to parse address", "address", sc.Address, "error", aErr)
			continue
		}

		hashBytes, _ := hex.DecodeString(sc.ProviderPublicKey)
		proof, vErr := w.providerClient.VerifyStorageADNLProof(ctx, hashBytes, addr)
		if vErr != nil {
			log.Error("failed to verify storage proof", "provider", sc.ProviderPublicKey, "error", vErr)
			providersIPs[sc.ProviderPublicKey] = db.ProviderIP{
				PublicKey: sc.ProviderPublicKey,
				IP:        "",
				Port:      0,
			}
			continue
		}

		l, _, dErr := w.dhtClient.FindAddresses(ctx, proof)
		if dErr != nil {
			log.Error("failed to find provider addresses", "provider", sc.ProviderPublicKey, "error", dErr)
			providersIPs[sc.ProviderPublicKey] = db.ProviderIP{
				PublicKey: sc.ProviderPublicKey,
				IP:        "",
				Port:      0,
			}
			continue
		}

		if l == nil {
			continue
		}

		providersIPs[sc.ProviderPublicKey] = db.ProviderIP{
			PublicKey: sc.ProviderPublicKey,
			IP:        l.Addresses[0].IP.String(),
			Port:      l.Addresses[0].Port,
		}
	}

	ips := make([]db.ProviderIP, 0, len(providersIPs))
	for _, p := range providersIPs {
		ips = append(ips, p)
	}
	err = w.providers.UpdateProvidersIPs(ctx, ips)
	if err != nil {
		log.Error("failed to add providers IPs", "error", err)
		interval = failureInterval
		return
	}

	contractProofsChecks := make([]db.ContractProofsCheck, 0, len(storageContracts))
	for _, sc := range storageContracts {
		// TODO: go routine batch
		reason := func() (reasonErr uint32) {
			addr, aErr := address.ParseAddr(sc.Address)
			if aErr != nil {
				log.Error("failed to parse address", "address", sc.Address, "error", aErr)
				reasonErr = 1 // "invalid address"
				return
			}

			prv, err := hex.DecodeString(sc.ProviderPublicKey)
			if err != nil || len(prv) != 32 {
				log.Error("failed to decode provider public key", "provider", sc.ProviderPublicKey, "error", err)
				reasonErr = 2 // "invalid provider public key"
				return
			}

			n, _ := rand.Int(rand.Reader, big.NewInt(int64(sc.Size)))
			toProof := n.Uint64()

			ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			stResp, err := w.providerClient.RequestStorageInfo(ctx, prv, addr, toProof)
			cancel()
			if err != nil {
				log.Error("failed to request storage info", "provider", sc.ProviderPublicKey, "error", err)
				reasonErr = 500 // "failed to request storage info"
				return
			}

			if stResp == nil || len(stResp.Proof) == 0 {
				log.Error("invalid storage proof response", "provider", sc.ProviderPublicKey, "response", stResp)
				reasonErr = 3 // "invalid storage proof response"
				return
			}

			return 0
		}()

		contractProofsChecks = append(contractProofsChecks, db.ContractProofsCheck{
			Address:         sc.Address,
			ProviderAddress: sc.ProviderAddress,
			Reason:          reason,
			Timestamp:       time.Now(),
		})
	}

	err = w.providers.UpdateContractProofsChecks(ctx, contractProofsChecks)
	if err != nil {
		log.Error("failed to update contract proofs checks", "error", err)
		interval = failureInterval
		return
	}

	log.Info("successfully updated contract proofs checks", "count", len(contractProofsChecks))

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
		masterAddr:     masterAddr,
		batchSize:      batchSize,
		logger:         logger,
	}
}
