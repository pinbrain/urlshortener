package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PgConfig struct {
	DSN string
}

type URLPgStore struct {
	pool *pgxpool.Pool
}

func NewPgDB(ctx context.Context, cfg PgConfig) (*URLPgStore, error) {
	pool, err := initPool(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize a db connection: %w", err)
	}
	return &URLPgStore{
		pool: pool,
	}, nil
}

func initPool(ctx context.Context, cfg PgConfig) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the DNS: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize a connection pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping the DB: %w", err)
	}
	return pool, nil
}

func (db *URLPgStore) Ping(ctx context.Context) error {
	err := db.pool.Ping(ctx)
	return err
}
