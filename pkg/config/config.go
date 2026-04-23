package config

import (
	"crypto/ed25519"
	"errors"
	"log/slog"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

var LogLevels = map[uint8]slog.Level{
	0: slog.LevelDebug,
	1: slog.LevelInfo,
	2: slog.LevelWarn,
	3: slog.LevelError,
}

type System struct {
	Port             string             `yaml:"system_port" env:"SYSTEM_PORT" envDefault:"9090"`
	ADNLPort         string             `yaml:"system_adnl_port" env:"SYSTEM_ADNL_PORT" envDefault:"16167"`
	AccessTokens     string             `env:"SYSTEM_ACCESS_TOKENS" envDefault:"dev-token"`
	Key              ed25519.PrivateKey `env:"SYSTEM_KEY" required:"false"`
	LogLevel         uint8              `yaml:"system_log_level" env:"SYSTEM_LOG_LEVEL" envDefault:"1"` // 0 - debug, 1 - info, 2 - warn, 3 - error
	StoreHistoryDays int                `yaml:"system_store_history_days" env:"SYSTEM_STORE_HISTORY_DAYS" envDefault:"90"`
}

type Metrics struct {
	Namespace          string `yaml:"namespace" env:"NAMESPACE" envDefault:"ton-storage"`
	ServerSubsystem    string `yaml:"server_subsystem" env:"SERVER_SUBSYSTEM" envDefault:"mtpo-server"`
	WorkersSubsystem   string `yaml:"workers_subsystem" env:"WORKERS_SUBSYSTEM" envDefault:"mtpo-workers"`
	DbSubsystem        string `yaml:"db_subsystem" env:"DB_SUBSYSTEM" envDefault:"mtpo-db"`
	ProvidersSubsystem string `yaml:"providers_subsystem" env:"PROVIDERS_SUBSYSTEM" envDefault:"mtpo-provider"`
}

type TON struct {
	MasterAddress string `env:"MASTER_ADDRESS" required:"true"`
	ConfigURL     string `yaml:"config_url" env:"TON_CONFIG_URL" required:"true"`
	BatchSize     uint32 `yaml:"batch_size" env:"BATCH_SIZE" required:"true" envDefault:"100"`
}

type Postgress struct {
	Host     string `yaml:"db_host" env:"DB_HOST" required:"true"`
	Port     string `yaml:"db_port" env:"DB_PORT" required:"true"`
	User     string `yaml:"db_user" env:"DB_USER" required:"true"`
	Password string `yaml:"db_password" env:"DB_PASSWORD" required:"true"`
	Name     string `yaml:"db_name" env:"DB_NAME" required:"true"`
}

type Config struct {
	System  System
	Metrics Metrics
	TON     TON
	DB      Postgress
}

func MustLoadConfig() *Config {
	configPath := os.Getenv("CONFIG_PATH")

	if configPath == "" {
		panic("config path is not set")
	}

	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		panic("config file does not exist")
	}

	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		panic("failed to read config: " + err.Error())
	}

	if cfg.System.Key == nil {
		_, priv, err := ed25519.GenerateKey(nil)
		if err != nil {
			panic("failed to generate private key: " + err.Error())
		}

		key := priv.Seed()
		cfg.System.Key = ed25519.NewKeyFromSeed(key)
	}

	return &cfg
}
