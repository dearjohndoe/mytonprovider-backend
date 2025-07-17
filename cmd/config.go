package main

import (
	"crypto/ed25519"
	"log"
	"log/slog"

	"github.com/caarlos0/env/v11"
)

var logLevels = map[uint8]slog.Level{
	0: slog.LevelDebug,
	1: slog.LevelInfo,
	2: slog.LevelWarn,
	3: slog.LevelError,
}

type System struct {
	Port             string             `env:"SYSTEM_PORT" envDefault:"9090"`
	ADNLPort         string             `env:"SYSTEM_ADNL_PORT" envDefault:"16167"`
	AccessTokens     string             `env:"SYSTEM_ACCESS_TOKENS" envDefault:""`
	Key              ed25519.PrivateKey `env:"SYSTEM_KEY" required:"false"`
	LogLevel         uint8              `env:"SYSTEM_LOG_LEVEL" envDefault:"1"` // 0 - debug, 1 - info, 2 - warn, 3 - error
	StoreHistoryDays int                `env:"SYSTEM_STORE_HISTORY_DAYS" envDefault:"90"`
}

type Metrics struct {
	Namespace        string `env:"NAMESPACE" default:"ton-storage"`
	ServerSubsystem  string `env:"SERVER_SUBSYSTEM" default:"mtpo-server"`
	WorkersSubsystem string `env:"WORKERS_SUBSYSTEM" default:"mtpo-workers"`
	DbSubsystem      string `env:"DB_SUBSYSTEM" default:"mtpo-db"`
}

type TON struct {
	MasterAddress string `env:"MASTER_ADDRESS" required:"true" envDefault:"UQB3d3d3d3d3d3d3d3d3d3d3d3d3d3d3d3d3d3d3d3d3d0x0"`
	ConfigURL     string `env:"TON_CONFIG_URL" required:"true" envDefault:"https://ton-blockchain.github.io/global.config.json"`
	BatchSize     uint32 `env:"BATCH_SIZE" required:"true" envDefault:"100"`
}

type Postgress struct {
	Host     string `env:"DB_HOST" required:"true"`
	Port     string `env:"DB_PORT" required:"true"`
	User     string `env:"DB_USER" required:"true"`
	Password string `env:"DB_PASSWORD" required:"true"`
	Name     string `env:"DB_NAME" required:"true"`
}

type Config struct {
	System  System
	Metrics Metrics
	TON     TON
	DB      Postgress
}

func loadConfig() *Config {
	cfg := &Config{}
	if err := env.Parse(&cfg.System); err != nil {
		log.Fatalf("Failed to parse system config: %v", err)
	}
	if err := env.Parse(&cfg.Metrics); err != nil {
		log.Fatalf("Failed to parse metrics config: %v", err)
	}
	if err := env.Parse(&cfg.DB); err != nil {
		log.Fatalf("Failed to parse db config: %v", err)
	}
	if err := env.Parse(&cfg.TON); err != nil {
		log.Fatalf("Failed to parse TON config: %v", err)
	}

	if cfg.System.Key == nil {
		_, priv, _ := ed25519.GenerateKey(nil)
		key := priv.Seed()
		cfg.System.Key = ed25519.NewKeyFromSeed(key)
	}

	return cfg
}
