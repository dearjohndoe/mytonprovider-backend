package tonclient

import "time"

type Transaction struct {
	Hash      []byte    `json:"hash"`
	LT        uint64    `json:"lt"`
	From      string    `json:"from"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}
