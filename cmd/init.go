package main

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xssnick/tonutils-go/adnl"
	"github.com/xssnick/tonutils-go/adnl/dht"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-storage-provider/pkg/transport"
)

func connectPostgres(ctx context.Context, config *Config, logger *slog.Logger) (connPool *pgxpool.Pool, err error) {
	cfg, err := newPostgresConfig(config, logger)
	if err != nil {
		return
	}

	connPool, err = pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		err = fmt.Errorf("failed to create a new Postgres connection pool: %w", err)
		return
	}

	connection, err := connPool.Acquire(ctx)
	if err != nil {
		err = fmt.Errorf("failed to acquire a connection from the Postgres pool: %w", err)
		return
	}
	defer connection.Release()

	err = connection.Ping(ctx)
	if err != nil {
		err = fmt.Errorf("failed to ping the Postgres database: %w", err)
		return
	}

	return
}

func newPostgresConfig(config *Config, logger *slog.Logger) (dbConfig *pgxpool.Config, err error) {
	const defaultMaxConns = int32(12)
	const defaultMinConns = int32(3)
	const defaultMaxConnLifetime = time.Hour
	const defaultMaxConnIdleTime = time.Minute * 30
	const defaultHealthCheckPeriod = time.Minute
	const defaultConnectTimeout = time.Second * 5
	const DATABASE_URL string = "postgres://%s:%s@%s:%s/%s"

	user := url.QueryEscape(config.DB.User)
	password := url.QueryEscape(config.DB.Password)

	pgUrl := fmt.Sprintf(DATABASE_URL, user, password, config.DB.Host, config.DB.Port, config.DB.Name)
	dbConfig, err = pgxpool.ParseConfig(pgUrl)
	if err != nil {
		err = fmt.Errorf("failed to parse Postgres connection string: %w", err)
		return
	}

	dbConfig.MaxConns = defaultMaxConns
	dbConfig.MinConns = defaultMinConns
	dbConfig.MaxConnLifetime = defaultMaxConnLifetime
	dbConfig.MaxConnIdleTime = defaultMaxConnIdleTime
	dbConfig.HealthCheckPeriod = defaultHealthCheckPeriod
	dbConfig.ConnConfig.ConnectTimeout = defaultConnectTimeout

	dbConfig.BeforeAcquire = func(ctx context.Context, c *pgx.Conn) bool {
		return true
	}

	dbConfig.AfterRelease = func(c *pgx.Conn) bool {
		return true
	}

	dbConfig.BeforeClose = func(c *pgx.Conn) {
		logger.Info("closed the connection pool to the database")
	}

	return
}

func newProviderClient(ctx context.Context, configURL, ADNLPort string, privateKey ed25519.PrivateKey) (dc *dht.Client, tc *transport.Client, err error) {
	lsCfg, err := liteclient.GetConfigFromUrl(ctx, configURL)
	if err != nil {
		err = fmt.Errorf("failed to get liteclient config: %w", err)
		return
	}

	_, dhtAdnlKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		err = fmt.Errorf("failed to generate DHT ADNL key: %w", err)
		return
	}

	dl, err := adnl.DefaultListener("0.0.0.0:" + ADNLPort)
	if err != nil {
		err = fmt.Errorf("failed to create default listener: %w", err)
		return
	}

	netMgr := adnl.NewMultiNetReader(dl)

	dhtGate := adnl.NewGatewayWithNetManager(dhtAdnlKey, netMgr)
	if err = dhtGate.StartClient(); err != nil {
		err = fmt.Errorf("failed to start DHT gateway: %w", err)
		return
	}

	dc, err = dht.NewClientFromConfig(dhtGate, lsCfg)
	if err != nil {
		err = fmt.Errorf("failed to create DHT client: %w", err)
		return
	}

	gateProvider := adnl.NewGatewayWithNetManager(privateKey, netMgr)
	if err = gateProvider.StartClient(); err != nil {
		err = fmt.Errorf("failed to start ADNL gateway for provider: %w", err)
		return
	}

	tc = transport.NewClient(gateProvider, dc)

	return
}
