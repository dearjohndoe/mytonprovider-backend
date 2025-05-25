package main

import (
	"log"

	"github.com/caarlos0/env/v11"
)

type System struct {
	Port string `env:"SYSTEM_PORT" envDefault:"9090"`
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
	if err := env.Parse(&cfg.TONStorage); err != nil {
		log.Fatalf("Failed to parse storage config: %v", err)
	}
	return cfg
}
