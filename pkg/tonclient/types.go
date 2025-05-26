package tonclient

import "time"

type Transaction struct {
	Hash         []byte    `json:"hash"`
	From         string    `json:"from"`
	Message      string    `json:"message"`
	RegisteredAt time.Time `json:"registered_at"`
}
