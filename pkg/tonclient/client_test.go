package tonclient

import (
	"context"
	"testing"
)

func Test_GetTransactions(t *testing.T) {
	ctx := context.Background()

	client, err := NewClient(ctx, "https://ton-blockchain.github.io/testnet-global.config.json")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	tx, err := client.GetTransactions(ctx, "UQB3d3d3d3d3d3d3d3d3d3d3d3d3d3d3d3d3d3d3d3d3d0x0", 5)
	if err != nil {
		t.Fatalf("GetTransactions failed: %v", err)
	}

	if len(tx) == 0 {
		t.Fatal("expected non-empty transaction list")
	}
}
