package tonclient

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
)

const (
	tspPrefix          = "tsp-"
	retries            = 20
	batch              = 30
	singleQueryTimeout = 5 * time.Second
)

type client struct {
	clientPool *liteclient.ConnectionPool
}

type Client interface {
	GetTransactions(ctx context.Context, addr string, lastProcessedLT uint64) (tx []*Transaction, err error)
}

// GetTransactions return all transactions between lastProcessedLT transaction and actual last transaction (both included)
// Not ordered by LT or other fileds.
func (c *client) GetTransactions(ctx context.Context, addr string, lastProcessedLT uint64) (txs []*Transaction, err error) {
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

func parseTx(tx *tlb.Transaction) (res *Transaction, ok bool) {
	in := tx.IO.In
	if in.MsgType != tlb.MsgTypeInternal {
		return
	}

	msg, ok := in.Msg.(*tlb.InternalMessage)
	if !ok {
		return
	}

	var comment string
	if msg.Body != nil {
		b := msg.Body.BeginParse()
		comment, _ = b.LoadStringSnake()
	}

	ok = true
	res = &Transaction{
		Hash:      tx.Hash,
		LT:        tx.LT,
		From:      msg.SrcAddr.String(),
		Message:   comment,
		CreatedAt: time.Unix(int64(msg.CreatedAt), 0),
	}

	return
}

func NewClient(ctx context.Context, configUrl string) (Client, error) {
	clientPool := liteclient.NewConnectionPool()

	err := clientPool.AddConnectionsFromConfigUrl(ctx, configUrl)
	if err != nil {
		panic(err)
	}

	return &client{
		clientPool: clientPool,
	}, nil
}
