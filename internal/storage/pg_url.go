package storage

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pinbrain/urlshortener/internal/logger"
	"github.com/pinbrain/urlshortener/internal/utils"
)

type PgConfig struct {
	DSN string
}

type URLPgStore struct {
	pool     *pgxpool.Pool
	urlDelCh chan urlDelBatchData
}

type urlDelBatchData struct {
	userID int
	urls   []string
}

const (
	delURLsBatchSize    = 100
	delURLBatchInterval = 10
)

func NewURLPgStore(ctx context.Context, cfg PgConfig) (*URLPgStore, error) {
	pool, err := initPool(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize a db connection: %w", err)
	}
	if err = initSchema(ctx, pool); err != nil {
		return nil, fmt.Errorf("failed to initialize a db scheme: %w", err)
	}
	store := &URLPgStore{
		pool:     pool,
		urlDelCh: make(chan urlDelBatchData, delURLsBatchSize),
	}

	go store.flushDelURLs(ctx)

	return store, nil
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
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx,
		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY
		);`,
	)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx,
		`CREATE TABLE IF NOT EXISTS shorten_urls (
			original VARCHAR(65536) NOT NULL UNIQUE,
			shorten VARCHAR(256) NOT NULL,
			user_id INT REFERENCES users (id),
			is_deleted BOOLEAN NOT NULL DEFAULT FALSE
		);`,
	)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (db *URLPgStore) flushDelURLs(ctx context.Context) {
	ticker := time.NewTicker(delURLBatchInterval * time.Second)
	defer ticker.Stop()

	var batch []urlDelBatchData

	for {
		select {
		case delURLs, ok := <-db.urlDelCh:
			if !ok {
				if len(batch) > 0 {
					db.executeDelBatch(ctx, batch)
					return
				}
			}
			batch = append(batch, delURLs)
			if len(batch) >= delURLsBatchSize {
				db.executeDelBatch(ctx, batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				db.executeDelBatch(ctx, batch)
				batch = batch[:0]
			}
		case <-ctx.Done():
			if len(batch) > 0 {
				db.executeDelBatch(ctx, batch)
			}
			return
		}
	}
}

func (db *URLPgStore) executeDelBatch(ctx context.Context, delBatch []urlDelBatchData) {
	batch := &pgx.Batch{}
	stmt := "UPDATE shorten_urls SET is_deleted = TRUE WHERE shorten = @shorten AND user_id = @user_id;"
	for _, del := range delBatch {
		for _, url := range del.urls {
			args := pgx.NamedArgs{
				"shorten": url,
				"user_id": del.userID,
			}
			batch.Queue(stmt, args)
		}
	}
	err := db.pool.SendBatch(ctx, batch).Close()
	if err != nil {
		logger.Log.Errorw("Error in batch deleting user URL", "err", err)
	}
}

func (db *URLPgStore) SaveURL(ctx context.Context, url string, userID int) (string, error) {
	id := utils.NewRandomString(urlIDLength)

	// оставляем возможность сохранять url неавторизованными пользователями
	var userIDValue interface{}
	if userID == 0 {
		userIDValue = nil
	} else {
		userIDValue = userID
	}

	_, err := db.pool.Exec(ctx,
		"INSERT INTO shorten_urls(original, shorten, user_id) VALUES($1, $2, $3);",
		url, id, userIDValue,
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

func (db *URLPgStore) SaveBatchURL(ctx context.Context, urls []ShortenURL, userID int) error {
	// оставляем возможность сохранять url неавторизованными пользователями
	var userIDValue interface{}
	if userID == 0 {
		userIDValue = nil
	} else {
		userIDValue = userID
	}
	batch := &pgx.Batch{}
	stmt := "INSERT INTO shorten_urls(original, shorten, user_id) VALUES(@original, @shorten, @user_id);"
	for i, url := range urls {
		id := utils.NewRandomString(urlIDLength)
		urls[i].Shorten = id
		args := pgx.NamedArgs{
			"original": url.Original,
			"shorten":  id,
			"user_id":  userIDValue,
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
		`SELECT original, is_deleted FROM shorten_urls WHERE shorten = $1`,
		id,
	)

	var url string
	var isDeleted bool
	if err := row.Scan(&url, &isDeleted); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("failed to select url from db: %w", err)
	}
	if isDeleted {
		return "", ErrIsDeleted
	}
	return url, nil
}

func (db *URLPgStore) CreateUser(ctx context.Context) (*User, error) {
	row := db.pool.QueryRow(ctx, "INSERT INTO users DEFAULT VALUES RETURNING id")
	var user User
	if err := row.Scan(&user.ID); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return &user, nil
}

func (db *URLPgStore) GetUser(ctx context.Context, id int) (*User, error) {
	if id <= 0 {
		return nil, errors.New("invalid user id")
	}
	row := db.pool.QueryRow(ctx,
		`SELECT id FROM users WHERE id = $1`,
		id,
	)
	var user User
	if err := row.Scan(&user.ID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoData
		}
		return nil, fmt.Errorf("failed to select user from db: %w", err)
	}
	return &user, nil
}

func (db *URLPgStore) GetUserURLs(ctx context.Context, userID int) ([]ShortenURL, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user id")
	}
	rows, err := db.pool.Query(ctx,
		`SELECT original, shorten FROM shorten_urls WHERE is_deleted = FALSE AND user_id = $1`,
		userID,
	)
	var userURLs []ShortenURL
	if err != nil {
		return nil, fmt.Errorf("failed to select user urls from db: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var shortenURL ShortenURL
		if err = rows.Scan(&shortenURL.Original, &shortenURL.Shorten); err != nil {
			return nil, fmt.Errorf("failed to read data from db url row: %w", err)
		}
		userURLs = append(userURLs, shortenURL)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to select user urls from db: %w", err)
	}
	return userURLs, nil
}

func (db *URLPgStore) DeleteUserURLs(userID int, urls []string) error {
	db.urlDelCh <- urlDelBatchData{userID: userID, urls: urls}
	return nil
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
	close(db.urlDelCh)
	return nil
}
