package providersmaster

import (
	"context"
	"encoding/hex"
	"math/big"
	"strings"
	"time"

	"github.com/xssnick/tonutils-storage-provider/pkg/transport"

	"mytonprovider-backend/pkg/models/db"
	"mytonprovider-backend/pkg/tonclient"
)

const (
	prefix                  = "tsp-"
	fakeSize                = 1
	providerResponseTimeout = 5 * time.Second
)

type providers interface {
	GetAllProvidersPubkeys(ctx context.Context) (pubkeys []string, err error)
	AddProviders(ctx context.Context, providers []db.ProviderInit) (err error)
	UpdateProviders(ctx context.Context, providers []db.ProviderInfo) (err error)
	DisableProviders(ctx context.Context, providers []string) (err error)
}

type ton interface {
	GetTransactions(ctx context.Context, addr string, count uint32) (tx []*tonclient.Transaction, err error)
}

type providersMasterWorker struct {
	providers      providers
	ton            ton
	providerClient *transport.Client
	masterAddr     string
	batchSize      uint32
}

type Worker interface {
	CollectNewProviders(ctx context.Context) (interval time.Duration, err error)
	UpdateKnownProviders(ctx context.Context) (interval time.Duration, err error)
}

func (w *providersMasterWorker) CollectNewProviders(ctx context.Context) (interval time.Duration, err error) {
	const (
		successInterval = 1 * time.Minute
		failureInterval = 5 * time.Second
	)

	interval = successInterval

	p, err := w.providers.GetAllProvidersPubkeys(ctx)
	if err != nil {
		interval = failureInterval
		return
	}

	knownProviders := make(map[string]struct{}, len(p))
	for _, pubkey := range p {
		knownProviders[strings.ToLower(pubkey)] = struct{}{}
	}

	txs, err := w.ton.GetTransactions(ctx, w.masterAddr, w.batchSize)
	if err != nil {
		interval = failureInterval
		return
	}

	uniqueProviders := make(map[string]db.ProviderInit)
	for i := range txs {
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

		// todo: store as bytearray
		prv, err := hex.DecodeString(pubkey)
		if err != nil || len(prv) != 32 {
			continue
		}

		uniqueProviders[pubkey] = db.ProviderInit{
			Pubkey:       pubkey,
			Address:      txs[i].From,
			RegisteredAt: txs[i].RegisteredAt,
		}
	}

	if len(uniqueProviders) == 0 {
		return
	}

	providersInit := make([]db.ProviderInit, 0, len(uniqueProviders))
	for _, provider := range uniqueProviders {
		providersInit = append(providersInit, provider)
	}

	err = w.providers.AddProviders(ctx, providersInit)
	if err != nil {
		interval = failureInterval
		return
	}

	return
}

func (w *providersMasterWorker) UpdateKnownProviders(ctx context.Context) (interval time.Duration, err error) {
	const (
		successInterval = 1 * time.Minute
		failureInterval = 5 * time.Second
	)

	interval = successInterval

	p, err := w.providers.GetAllProvidersPubkeys(ctx)
	if err != nil {
		interval = failureInterval
		return
	}

	if len(p) == 0 {
		return
	}

	providersInfo := make([]db.ProviderInfo, 0, len(p))
	disabledProviders := make([]string, 0, len(p))
	for _, pubkey := range p {
		prv, err := hex.DecodeString(pubkey)
		if err != nil || len(prv) != 32 {
			continue
		}

		timeoutCtx, cancel := context.WithTimeout(ctx, providerResponseTimeout)
		defer cancel()

		rates, err := w.providerClient.GetStorageRates(timeoutCtx, prv, fakeSize)
		if err != nil {
			disabledProviders = append(disabledProviders, pubkey)
			continue
		}

		providersInfo = append(providersInfo, db.ProviderInfo{
			Pubkey:       pubkey,
			RatePerMBDay: new(big.Int).SetBytes(rates.RatePerMBDay).Int64(),
			MinBounty:    new(big.Int).SetBytes(rates.MinBounty).Int64(),
			MinSpan:      rates.MinSpan,
			MaxSpan:      rates.MaxSpan,
		})
	}

	err = w.providers.DisableProviders(ctx, disabledProviders)
	if err != nil {
		interval = failureInterval
		return
	}

	err = w.providers.UpdateProviders(ctx, providersInfo)
	if err != nil {
		interval = failureInterval
		return
	}

	return
}

func NewWorker(
	providers providers,
	ton ton,
	providerClient *transport.Client,
	masterAddr string,
	batchSize uint32,
) Worker {
	return &providersMasterWorker{
		providers:      providers,
		ton:            ton,
		providerClient: providerClient,
		masterAddr:     masterAddr,
		batchSize:      batchSize,
	}
}
