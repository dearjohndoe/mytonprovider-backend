package tonclient

import (
	"context"
	"log"
	"math/big"
	"time"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"

	"mytonprovider-backend/pkg/models"
)

const (
	tspPrefix = "tsp-"
	retries   = 20
)

type client struct {
	clientPool *liteclient.ConnectionPool
	minAmount  *big.Int
}

type Client interface {
	GetTransactions(ctx context.Context, addr string, count uint32) (tx []*Transaction, err error)
}

func (c *client) GetTransactions(ctx context.Context, addr string, count uint32) (txs []*Transaction, err error) {
	api := ton.NewAPIClient(c.clientPool).WithRetry(retries)
	a, _ := address.ParseAddr(addr)
	b, err := api.GetMasterchainInfo(ctx)
	if err != nil {
		log.Printf("get masterchain info err: %s", err.Error())
		err = &models.AppError{
			Code:    models.InternalServerErrorCode,
			Message: "network error",
		}
		return
	}

	account, err := api.GetAccount(ctx, b, a)
	if err != nil {
		log.Printf("get account err: %s", err.Error())
		err = &models.AppError{
			Code:    models.InternalServerErrorCode,
			Message: "network error",
		}

		return
	}

	list, err := api.ListTransactions(ctx, account.State.Address, count, account.LastTxLT, account.LastTxHash)
	if err != nil {
		log.Printf("list transactions err: %s", err.Error())
		err = &models.AppError{
			Code:    models.InternalServerErrorCode,
			Message: "network error",
		}

		return
	}

	txs = make([]*Transaction, 0, len(list))
	for _, t := range list {
		in := t.IO.In
		if in.MsgType != tlb.MsgTypeInternal {
			continue
		}

		msg, ok := in.Msg.(*tlb.InternalMessage)
		if !ok {
			continue
		}

		if msg.Body == nil {
			continue
		}

		b := msg.Body.BeginParse()
		comment, err := b.LoadStringSnake()
		if err != nil {
			continue
		}

		if len(comment) == 0 {
			continue
		}

		// TODO: check amount?

		txs = append(txs, &Transaction{
			Hash:         t.Hash,
			From:         msg.SrcAddr.String(),
			Message:      comment,
			RegisteredAt: time.Now(), // TODO: get from transaction
		})
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
