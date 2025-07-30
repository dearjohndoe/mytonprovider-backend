package ifconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

type client struct {
	logger *slog.Logger
}

type IFConfig interface {
	GetIPInfo(ctx context.Context, ip string) (conf *Info, err error)
}

func (c *client) GetIPInfo(ctx context.Context, ip string) (*Info, error) {
	log := c.logger.With("method", "GetIPInfo")

	req, err := http.NewRequestWithContext(ctx, "GET", "https://ifconfig.co/json?ip="+ip, nil)
	if err != nil {
		log.Error("failed to create request", "error", err)
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error("failed to execute request", "error", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Error("unexpected response status", "status", resp.Status)
		return nil, fmt.Errorf("failed to get IP config: %s", resp.Status)
	}

	var config Info
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		log.Error("failed to decode response", "error", err)
		return nil, err
	}

	return &config, nil
}

func NewClient(logger *slog.Logger) IFConfig {
	return &client{
		logger: logger,
	}
}
