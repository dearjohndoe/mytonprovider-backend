package providersmaster

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
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
	"github.com/xssnick/tonutils-go/adnl/dht"
	"github.com/xssnick/tonutils-go/adnl/keys"
	"github.com/xssnick/tonutils-go/adnl/overlay"
	"github.com/xssnick/tonutils-go/adnl/rldp"
	"github.com/xssnick/tonutils-go/tl"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/tvm/cell"
	"github.com/xssnick/tonutils-storage-provider/pkg/transport"
	"github.com/xssnick/tonutils-storage/storage"

	"mytonprovider-backend/pkg/clients/ifconfig"
	tonclient "mytonprovider-backend/pkg/clients/ton"
	"mytonprovider-backend/pkg/constants"
	"mytonprovider-backend/pkg/models/db"
	"mytonprovider-backend/pkg/utils"
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
	prv            ed25519.PrivateKey
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

	// todo: перепроверь что всё ещё работает и накидай тестов
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

// updateActiveContracts check storage proofs for all bags and update status for relations provider-contract
func (w *providersMasterWorker) updateActiveContracts(ctx context.Context, storageContracts []db.ContractToProviderRelation, availableProvidersIPs map[string]db.ProviderIP) (err error) {
	log := w.logger.With(slog.String("worker", "StoreProof"), slog.String("function", "updateActiveContracts"))

	if len(storageContracts) == 0 || len(availableProvidersIPs) == 0 {
		return nil
	}

	providersContracts := make(map[string][]db.ContractToProviderRelation)
	for _, sc := range storageContracts {
		providersContracts[sc.ProviderPublicKey] = append(providersContracts[sc.ProviderPublicKey], sc)
	}

	wg := sync.WaitGroup{}
	semaphore := make(chan struct{}, maxConcurrentBagChecks)
	var bagsStatuses sync.Map

	for pubkey, contracts := range providersContracts {
		wg.Add(1)

		go func(pubkey string, contracts []db.ContractToProviderRelation) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			gw := adnl.NewGateway(w.prv)
			defer gw.Close()

			if err = gw.StartClient(); err != nil {
				log.Error("failed to start ADNL gateway", "error", err)
				return
			}

			ip, ok := availableProvidersIPs[pubkey]
			if !ok {
				fillStatuses(&bagsStatuses, contracts, constants.IPNotFound)
				return
			}

			checkProviderFiles(ctx, gw, ip, contracts, &bagsStatuses, log)
		}(pubkey, contracts)
	}

	wg.Wait()

	valid := 0
	contractProofsChecks := make([]db.ContractProofsCheck, 0, len(storageContracts))
	bagsStatuses.Range(func(_, value any) (resp bool) {
		resp = true

		proof, ok := value.(db.ContractProofsCheck)
		if !ok {
			return
		}

		contractProofsChecks = append(contractProofsChecks, proof)

		if proof.Reason == constants.ValidStorageProof {
			valid++
		}

		return
	})

	err = w.providers.UpdateContractProofsChecks(ctx, contractProofsChecks)
	if err != nil {
		log.Error("failed to update contract proofs checks", "error", err)
		return
	}

	log.Info("successfully updated contract proofs checks", "count", len(contractProofsChecks), "valid", valid)

	return nil
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
	maxFailureThreshold := uint32(len(storageContracts) / 100 * 20)
	var failsInARow uint32

	addr := ip.Storage.IP + ":" + strconv.Itoa(int(ip.Storage.Port))
	peer, rErr := gw.RegisterClient(addr, ip.Storage.PublicKey)
	if rErr != nil {
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

		// todo: remove
		log.Debug("bag checked", "bag_id", sc.BagID, "reason_status", reason)

		if reason == constants.ValidStorageProof {
			failsInARow = 0
		} else {
			failsInARow++
		}
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
		reason = constants.UnavailableProvider
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
	var res storage.TorrentInfoContainer
	rlCtx, rlc := context.WithTimeout(ctx, rlQueryTimeout)
	err = rl.DoQuery(rlCtx, 32<<20, overlay.WrapQuery(over, &storage.GetTorrentInfo{}), &res)
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

	var info storage.TorrentInfo
	err = tlb.LoadFromCell(&info, cl.BeginParse())
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
	var piece storage.Piece
	rl2Ctx, rl2c := context.WithTimeout(ctx, rlQueryTimeout)
	err = rl.DoQuery(rl2Ctx, 32<<20, overlay.WrapQuery(over, &storage.GetPiece{PieceID: pieceID}), &piece)
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

			providerIPs, pErr := w.processProviderIPs(ctx, contract, log)
			if pErr != nil {
				return
			}

			mu.Lock()
			availableProvidersIPs[contract.ProviderPublicKey] = providerIPs
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

func (w *providersMasterWorker) processProviderIPs(ctx context.Context, sc db.ContractToProviderRelation, log *slog.Logger) (result db.ProviderIP, err error) {
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

	result.Storage, err = w.findStorageIP(ctx, addr, pk)
	if err != nil {
		log.Error("failed to find storage IP", "address", sc.Address, "error", err)
		return
	}

	result.Provider, err = w.findProviderIP(ctx, pk)
	if err != nil {
		log.Error("failed to find provider IP", "error", err)
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
		err = fmt.Errorf("failed to verify storage adnl proof: %w", err)
		return
	}

	dhtTimeoutCtx, cancel := context.WithTimeout(ctx, dhtTimeout)
	defer cancel()
	l, pub, err := w.dhtClient.FindAddresses(dhtTimeoutCtx, proof)
	if err != nil {
		err = fmt.Errorf("failed to find addresses in dht: %w", err)
		return
	}

	if l == nil || len(l.Addresses) == 0 {
		err = fmt.Errorf("no storage addresses found")
		return
	}

	ip.PublicKey = pub
	ip.IP = l.Addresses[0].IP.String()
	ip.Port = l.Addresses[0].Port

	return
}

func (w *providersMasterWorker) findProviderIP(ctx context.Context, pk []byte) (ip db.IPInfo, err error) {
	channelKeyId, err := tl.Hash(keys.PublicKeyED25519{Key: pk})
	if err != nil {
		err = fmt.Errorf("failed to calc hash of provider key: %w", err)
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
		err = fmt.Errorf("failed to find storage-provider in dht: %w", err)
		return
	}

	var nodeAddr transport.ProviderDHTRecord
	if _, pErr := tl.Parse(&nodeAddr, dhtVal.Data, true); pErr != nil {
		err = fmt.Errorf("failed to parse node dht value: %w", pErr)
		return
	}

	if len(nodeAddr.ADNLAddr) == 0 {
		err = fmt.Errorf("no adnl addresses in node dht value")
		return
	}

	dhtTimeoutCtx2, cancel2 := context.WithTimeout(ctx, dhtTimeout)
	defer cancel2()
	addrList, pubKey, fErr := w.dhtClient.FindAddresses(dhtTimeoutCtx2, nodeAddr.ADNLAddr)
	if fErr != nil {
		err = fmt.Errorf("failed to find adnl addresses in dht: %w", fErr)
		return
	}

	for _, addr := range addrList.Addresses {
		ip, found := func() (ip db.IPInfo, found bool) {
			a := addr.IP.String() + ":" + fmt.Sprint(addr.Port)

			gw := adnl.NewGateway(w.prv)
			defer gw.Close()

			if cErr := gw.StartClient(); cErr != nil {
				err = fmt.Errorf("failed to start ADNL gateway: %w", cErr)
				return
			}

			peer, rErr := gw.RegisterClient(a, pubKey)
			if rErr != nil {
				err = fmt.Errorf("failed to register peer: %w", rErr)
				return
			}

			timeoutCtx, cancel := context.WithTimeout(ctx, pingTimeout)
			_, pErr := peer.Ping(timeoutCtx)
			cancel()
			if pErr != nil {
				err = fmt.Errorf("ping failed: %w", pErr)
				return
			}

			ip.IP = addr.IP.String()
			ip.Port = addr.Port
			found = true

			return
		}()

		if found {
			return ip, nil
		}
	}

	err = fmt.Errorf("no reachable adnl addresses found")

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
	masterAddr string,
	batchSize uint32,
	logger *slog.Logger,
) Worker {
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
		masterAddr:     masterAddr,
		batchSize:      batchSize,
		logger:         logger,
	}
}
