package ifconfig

import (
	"context"
	"log/slog"
	"testing"
)

func Test_GetIPInfo(t *testing.T) {
	ctx := context.Background()

	logger := slog.Default()

	client := NewClient(logger)
	ip := "8.8.8.8"

	conf, err := client.GetIPInfo(ctx, ip)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if conf == nil {
		t.Fatal("expected non-nil config")
	}
}
