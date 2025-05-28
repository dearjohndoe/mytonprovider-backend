package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func connectPostgres(config *Config, logger *log.Logger) (connPool *pgxpool.Pool, err error) {
	cfg, err := newPostgresConfig(config, logger)
	if err != nil {
		return
	}

	connPool, err = pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		err = fmt.Errorf("failed to create a new Postgres connection pool: %w", err)
		return
	}

	connection, err := connPool.Acquire(context.Background())
	if err != nil {
		err = fmt.Errorf("failed to acquire a connection from the Postgres pool: %w", err)
		return
	}
	defer connection.Release()

	err = connection.Ping(context.Background())
	if err != nil {
		err = fmt.Errorf("failed to ping the Postgres database: %w", err)
		return
	}

	return
}

func newPostgresConfig(config *Config, logger *log.Logger) (dbConfig *pgxpool.Config, err error) {
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
		logger.Println("Closed the connection pool to the database!")
	}

	return
}
