package config

import (
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

var LogLevels = map[uint8]slog.Level{
	0: slog.LevelDebug,
	1: slog.LevelInfo,
	2: slog.LevelWarn,
	3: slog.LevelError,
}

type System struct {
	Port             string `yaml:"system_port" env:"SYSTEM_PORT" envDefault:"9090"`
	AccessTokens     string `env:"SYSTEM_ACCESS_TOKENS" envDefault:"dev-token"`
	LogLevel         uint8  `yaml:"system_log_level" env:"SYSTEM_LOG_LEVEL" envDefault:"1"` // 0 - debug, 1 - info, 2 - warn, 3 - error
	StoreHistoryDays int    `yaml:"system_store_history_days" env:"SYSTEM_STORE_HISTORY_DAYS" envDefault:"90"`
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
}

type Postgress struct {
	Host     string `yaml:"db_host" env:"DB_HOST" required:"true"`
	Port     string `yaml:"db_port" env:"DB_PORT" required:"true"`
	User     string `yaml:"db_user" env:"DB_USER" required:"true"`
	Password string `yaml:"db_password" env:"DB_PASSWORD" required:"true"`
	Name     string `yaml:"db_name" env:"DB_NAME" required:"true"`
}

type Redis struct {
	Addr         string `yaml:"addr"          env:"REDIS_ADDR"          envDefault:"redis:6379"`
	Password     string `                     env:"REDIS_PASSWORD"     envDefault:""`
	DB           int    `yaml:"db"            env:"REDIS_DB"            envDefault:"0"`
	Group        string `yaml:"group"         env:"REDIS_GROUP"         envDefault:"mtpa-backend"`
	StreamPrefix string `yaml:"stream_prefix" env:"REDIS_STREAM_PREFIX" envDefault:"mtpa"`
	TriggerMaxLen int64 `yaml:"trigger_maxlen"                          envDefault:"10000"`
	BlockMs      int    `yaml:"block_ms"                                envDefault:"5000"`
	Parallel     int    `yaml:"parallel"                                envDefault:"1"`
}

// CycleSchedule — расписание одного цикла.
type CycleSchedule struct {
	Enabled        bool          `yaml:"enabled"`
	Interval       time.Duration `yaml:"interval"`
	SingleInflight bool          `yaml:"single_inflight"`
}

type Cycles struct {
	ScanMaster       CycleSchedule `yaml:"scan_master"`
	ScanWallets      CycleSchedule `yaml:"scan_wallets"`
	ResolveEndpoints CycleSchedule `yaml:"resolve_endpoints"`
	ProbeRates       CycleSchedule `yaml:"probe_rates"`
	InspectContracts CycleSchedule `yaml:"inspect_contracts"`
	CheckProofs      CycleSchedule `yaml:"check_proofs"`
	LookupIPInfo     CycleSchedule `yaml:"lookup_ipinfo"`
}

type Config struct {
	System  System
	Metrics Metrics
	TON     TON
	DB      Postgress
	Redis   Redis  `yaml:"redis"`
	Cycles  Cycles `yaml:"cycles"`
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

	return &cfg
}
