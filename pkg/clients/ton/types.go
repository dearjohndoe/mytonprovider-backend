package tonclient

import (
	"time"
)

type Transaction struct {
	Hash      []byte    `json:"hash"`
	LT        uint64    `json:"lt"`
	Op        uint64    `json:"op"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type StorageContract struct {
	Address   string
	BagID     string
	OwnerAddr string
	Size      uint64
	ChunkSize uint64
}

type StorageContractProviders struct {
	Address         string
	Balance         uint64
	Providers       []Provider
	LiteServerError bool
}

type Provider struct {
	Key           string
	LastProofTime time.Time
	RatePerMBDay  uint64
	MaxSpan       uint32
}
