package storage

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pinbrain/urlshortener/internal/utils"
)

type PgConfig struct {
	DSN string
}

type URLPgStore struct {
	pool *pgxpool.Pool
}

func NewURLPgStore(ctx context.Context, cfg PgConfig) (*URLPgStore, error) {
	pool, err := initPool(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize a db connection: %w", err)
	}
	if err = initSchema(ctx, pool); err != nil {
		return nil, fmt.Errorf("failed to initialize a db scheme: %w", err)
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
	if err = pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping the DB: %w", err)
	}
	return pool, nil
}

func initSchema(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx,
		`CREATE TABLE IF NOT EXISTS shorten_urls (
			original VARCHAR(65536) NOT NULL UNIQUE,
			shorten VARCHAR(256) NOT NULL
		);`,
	)
	if err != nil {
		return err
	}
	return nil
}

func (db *URLPgStore) SaveURL(ctx context.Context, url string) (string, error) {
	id := utils.NewRandomString(urlIDLength)
	_, err := db.pool.Exec(ctx,
		"INSERT INTO shorten_urls(original, shorten) VALUES($1, $2);",
		url, id,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			row := db.pool.QueryRow(ctx,
				`SELECT shorten FROM shorten_urls WHERE original = $1`,
				url,
			)
			if err = row.Scan(&id); err != nil {
				return "", fmt.Errorf("failed to select existing url from db after unique conflict: %w", err)
			}
			return id, ErrConflict
		}
		return "", fmt.Errorf("failed to insert record to db: %w", err)
	}
	return id, nil
}

func (db *URLPgStore) SaveBatchURL(ctx context.Context, urls []ShortenURL) error {
	batch := &pgx.Batch{}
	stmt := "INSERT INTO shorten_urls(original, shorten) VALUES(@original, @shorten);"
	for i, url := range urls {
		id := utils.NewRandomString(urlIDLength)
		urls[i].Shorten = id
		args := pgx.NamedArgs{
			"original": url.Original,
			"shorten":  id,
		}
		batch.Queue(stmt, args)
	}
	err := db.pool.SendBatch(ctx, batch).Close()
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return ErrConflict
		}
		return fmt.Errorf("failed to save batch of urls: %w", err)
	}

	return nil
}

func (db *URLPgStore) GetURL(ctx context.Context, id string) (string, error) {
	row := db.pool.QueryRow(ctx,
		`SELECT original FROM shorten_urls WHERE shorten = $1`,
		id,
	)

	var url string
	if err := row.Scan(&url); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("failed to select url from db: %w", err)
	}
	return url, nil
}

func (db *URLPgStore) IsValidID(id string) bool {
	regStr := fmt.Sprintf(`^[a-zA-Z0-9]{%d}$`, urlIDLength)
	validIDReg := regexp.MustCompile(regStr)
	return validIDReg.MatchString(id)
}

func (db *URLPgStore) Ping(ctx context.Context) error {
	err := db.pool.Ping(ctx)
	return err
}

func (db *URLPgStore) Close() error {
	db.pool.Close()
	return nil
}
