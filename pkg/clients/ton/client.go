package tonclient

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	pContract "github.com/xssnick/tonutils-storage-provider/pkg/contract"

	"mytonprovider-backend/pkg/utils"
)

const (
	getProvidersRetries = 5
	retries             = 20
	parrallelRequests   = 30
	batch               = 100
	singleQueryTimeout  = 5 * time.Second
)

type client struct {
	clientPool *liteclient.ConnectionPool
	logger     *slog.Logger
}

type Client interface {
	GetTransactions(ctx context.Context, addr string, lastProcessedLT uint64) (tx []*Transaction, err error)
	GetStorageContractsInfo(ctx context.Context, addrs []string) (contracts []StorageContract, err error)
	GetProvidersInfo(ctx context.Context, addrs []string) (contractsProviders []StorageContractProviders, err error)
}

// GetTransactions return all transactions between lastProcessedLT transaction and actual last transaction (both included)
// Not ordered by LT or other fileds, it gets them by batches from lastProcessedLT and newest(or deadline exceeded)
func (c *client) GetTransactions(ctx context.Context, addr string, lastProcessedLT uint64) (txs []*Transaction, err error) {
	log := c.logger.With("method", "GetTransactions")
	api := ton.NewAPIClient(c.clientPool).WithTimeout(singleQueryTimeout).WithRetry(retries)
	a, _ := address.ParseAddr(addr)
	block, err := api.GetMasterchainInfo(ctx)
	if err != nil {
		err = fmt.Errorf("get masterchain info err: %w", err)
		return
	}

	account, err := api.GetAccount(ctx, block, a)
	if err != nil {
		err = fmt.Errorf("get account err: %w", err)
		return
	}

	lastLT, lastHash := account.LastTxLT, account.LastTxHash
	var transactions []*tlb.Transaction
list:
	for {
		res, errTx := api.ListTransactions(ctx, a, batch, lastLT, lastHash)
		if errTx != nil {
			if errors.Is(errTx, ton.ErrNoTransactionsWereFound) && (len(transactions) > 0) {
				break
			}

			if errors.Is(errTx, context.DeadlineExceeded) {
				// No need to collect all transactions, if deadline, then we have enough
				// We collect more next time
				log.Info("just got deadline exceeded, stopping transaction collection", "collected", len(transactions), "lastLT", lastLT)
				break
			}

			v, ok := errTx.(ton.LSError)
			if ok && v.Code == -400 {
				log.Info("just got -400 error, stopping transaction collection", "collected", len(transactions), "addr", addr, "lastLT", lastLT)
				break
			}

			err = fmt.Errorf("list transactions: %w", errTx)
			return
		}

		if len(res) == 0 {
			break
		}

		for i := range res {
			reverseIter := len(res) - 1 - i
			tx := res[reverseIter]
			if tx.LT <= lastProcessedLT {
				transactions = append(transactions, res[reverseIter:]...)
				break list
			}
		}

		lastLT, lastHash = res[0].PrevTxLT, res[0].PrevTxHash
		transactions = append(transactions, res...)
	}

	txs = make([]*Transaction, 0, len(transactions))
	for _, t := range transactions {
		if tx, ok := parseTx(t); ok {
			txs = append(txs, tx)
		}
	}

	return
}

// GetStorageContractsInfo interacts with storage contracts to get their info
func (c *client) GetStorageContractsInfo(ctx context.Context, addrs []string) (contracts []StorageContract, err error) {
	log := c.logger.With("method", "GetStorageContractsInfo")
	api := ton.NewAPIClient(c.clientPool).WithTimeout(singleQueryTimeout).WithRetry(retries)
	block, err := api.GetMasterchainInfo(ctx)
	if err != nil {
		err = fmt.Errorf("get masterchain info err: %w", err)
		return
	}

	contracts = make([]StorageContract, 0, len(addrs))
	for _, a := range addrs {
		addr, err := address.ParseAddr(a)
		if err != nil {
			log.Error("invalid address", slog.String("address", a), slog.String("error", err.Error()))
			continue
		}

		info, err := pContract.GetStorageInfoV1(ctx, api, block, addr)
		if err != nil {
			log.Error("get storage info", slog.String("address", a), slog.String("error", err.Error()))
			continue
		}

		if info == nil {
			log.Error("storage contract not found", slog.String("address", a))
			continue
		}

		contracts = append(contracts, StorageContract{
			Address:   a,
			BagID:     fmt.Sprintf("%x", info.TorrentHash),
			OwnerAddr: info.OwnerAddr.String(),
			Size:      info.Size,
			ChunkSize: info.ChunkSize,
		})
	}

	return
}

func (c *client) GetProvidersInfo(ctx context.Context, addrs []string) (contractsProviders []StorageContractProviders, err error) {
	log := c.logger.With("method", "GetProvidersInfo")
	api := ton.NewAPIClient(c.clientPool).WithTimeout(singleQueryTimeout).WithRetry(retries)
	block, err := api.GetMasterchainInfo(ctx)
	if err != nil {
		err = fmt.Errorf("get masterchain info err: %w", err)
		return
	}

	var semaphore = make(chan struct{}, parrallelRequests)
	var wg sync.WaitGroup
	var mu sync.Mutex

	contractsProviders = make([]StorageContractProviders, 0, len(addrs))
	for _, a := range addrs {
		wg.Add(1)

		go func(addrStr string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			addr, parseErr := address.ParseAddr(addrStr)
			if parseErr != nil {
				log.Error("invalid address", slog.String("address", addrStr), slog.String("error", parseErr.Error()))
				return
			}

			var info []pContract.ProviderDataV1
			var coins tlb.Coins
			callErr := utils.TryNTimes(func() error {
				var cErr error
				info, coins, cErr = pContract.GetProvidersV1(ctx, api, block, addr)
				return cErr
			}, getProvidersRetries)
			if callErr != nil {
				log.Error("get providers info", slog.String("address", addrStr), slog.String("error", callErr.Error()))
			}

			providers := make([]Provider, 0, len(info))
			for _, p := range info {
				providers = append(providers, Provider{
					Key:           string(p.Key),
					LastProofTime: p.LastProofAt,
					RatePerMBDay:  p.RatePerMB.Nano().Uint64(),
					MaxSpan:       p.MaxSpan,
				})
			}

			mu.Lock()
			contractsProviders = append(contractsProviders, StorageContractProviders{
				Address:         addrStr,
				Balance:         coins.Nano().Uint64(),
				Providers:       providers,
				LiteServerError: callErr != nil,
			})
			mu.Unlock()
		}(a)
	}

	wg.Wait()

	return
}

func parseTx(tx *tlb.Transaction) (res *Transaction, ok bool) {
	var comment, srcAddr, dstAddr string
	var op uint64
	var createdAt int64

	in := tx.IO.In
	switch in.MsgType {
	case tlb.MsgTypeInternal:
		{
			var msg *tlb.InternalMessage
			msg, ok = in.Msg.(*tlb.InternalMessage)
			if !ok {
				return
			}

			srcAddr = msg.SrcAddr.String()
			dstAddr = msg.DstAddr.String()
			createdAt = int64(msg.CreatedAt)

			if msg.Payload() != nil {
				{
					b := msg.Payload().BeginParse()
					comment, _ = b.LoadStringSnake()
				}
				{
					b := msg.Payload().BeginParse()
					op, _ = b.LoadUInt(32)
				}
			}
		}
	default:
		{
			return
		}
	}

	ok = true
	res = &Transaction{
		Hash:      tx.Hash,
		LT:        tx.LT,
		Op:        op,
		From:      srcAddr,
		To:        dstAddr,
		Message:   comment,
		CreatedAt: time.Unix(createdAt, 0),
	}

	return
}

func NewClient(ctx context.Context, configUrl string, logger *slog.Logger) (Client, error) {
	clientPool := liteclient.NewConnectionPool()

	err := clientPool.AddConnectionsFromConfigUrl(ctx, configUrl)
	if err != nil {
		panic(err)
	}

	return &client{
		clientPool: clientPool,
		logger:     logger,
	}, nil
}
