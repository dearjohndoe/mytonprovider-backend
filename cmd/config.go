package main

import (
	"log"

	"github.com/caarlos0/env/v11"
)

type System struct {
	Port         string `env:"SYSTEM_PORT" envDefault:"9090"`
	AccessTokens string `env:"SYSTEM_ACCESS_TOKENS" envDefault:""`
}

type TON struct {
	MasterAddress string `env:"MASTER_ADDRESS" required:"true" envDefault:"UQB3d3d3d3d3d3d3d3d3d3d3d3d3d3d3d3d3d3d3d3d3d0x0"`
	ConfigURL     string `env:"TON_CONFIG_URL" required:"true" envDefault:"https://ton.org/global-config.json"`
	BatchSize     uint32 `env:"BATCH_SIZE" required:"true" envDefault:"100"`
}

type Postgress struct {
	Host     string `env:"DB_HOST" required:"true"`
	Port     string `env:"DB_PORT" required:"true"`
	User     string `env:"DB_USER" required:"true"`
	Password string `env:"DB_PASSWORD" required:"true"`
	Name     string `env:"DB_NAME" required:"true"`
}

type TONStorage struct {
	Password string `env:"TON_STORAGE_PASSWORD" required:"true"`
	Username string `env:"TON_STORAGE_USERNAME" required:"true"`
	Host     string `env:"TON_STORAGE_HOST" required:"true"`
	Port     string `env:"TON_STORAGE_PORT" required:"true"`
}

type Config struct {
	System     System
	TON        TON
	DB         Postgress
	TONStorage TONStorage
}

func loadConfig() *Config {
	cfg := &Config{}
	if err := env.Parse(&cfg.System); err != nil {
		log.Fatalf("Failed to parse system config: %v", err)
	}
	if err := env.Parse(&cfg.DB); err != nil {
		log.Fatalf("Failed to parse db config: %v", err)
	}
	if err := env.Parse(&cfg.TON); err != nil {
		log.Fatalf("Failed to parse TON config: %v", err)
	}
	if err := env.Parse(&cfg.TONStorage); err != nil {
		log.Fatalf("Failed to parse storage config: %v", err)
	}

	return cfg
}
