package providersmaster

import (
	"context"
	"fmt"
	"strings"
	"time"

	"mytonprovider-backend/pkg/models/db"
	"mytonprovider-backend/pkg/tonclient"
)

const (
	prefix = "tsp-"
)

type providers interface {
	GetAllProvidersPubkeys(ctx context.Context) (pubkeys []string, err error)
	AddProviders(ctx context.Context, providers []db.ProviderInit) (err error)
}

type ton interface {
	GetTransactions(ctx context.Context, addr string, count uint32) (tx []*tonclient.Transaction, err error)
}

type providersMasterWorker struct {
	providers  providers
	ton        ton
	masterAddr string
	batchSize  uint32
}

type Worker interface {
	CollectNewProviders(ctx context.Context) (interval time.Duration, err error)
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

	newProviders := []db.ProviderInit{}
	for i := range txs {
		pos := strings.Index(txs[i].Message, prefix)
		if pos < 0 {
			continue
		}

		pos += len(prefix)
		if pos >= len(txs[i].Message) {
			continue
		}

		publicKey := strings.ToLower(txs[i].Message[pos:])

		fmt.Println(publicKey)

		if _, ok := knownProviders[publicKey]; ok {
			continue
		}

		newProviders = append(newProviders, db.ProviderInit{
			Pubkey:       publicKey,
			Address:      txs[i].From,
			RegisteredAt: txs[i].RegisteredAt,
		})
	}

	if len(newProviders) == 0 {
		return
	}

	err = w.providers.AddProviders(ctx, newProviders)
	if err != nil {
		interval = failureInterval
		return
	}

	return
}

func NewWorker(
	providers providers,
	ton ton,
	masterAddr string,
	batchSize uint32,
) Worker {
	return &providersMasterWorker{
		providers:  providers,
		ton:        ton,
		masterAddr: masterAddr,
		batchSize:  batchSize,
	}
}
