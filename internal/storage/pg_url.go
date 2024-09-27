package storage

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pinbrain/urlshortener/internal/logger"
	"github.com/pinbrain/urlshortener/internal/utils"
)

// PgConfig описывает структуру конфигурации БД.
type PgConfig struct {
	DSN string
}

// PgxPoolI описывает интерфейс Pool postgresql. Совместим с моком для тестов.
type PgxPoolI interface {
	Begin(context.Context) (pgx.Tx, error)
	Close()
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Ping(ctx context.Context) error
}

// URLPgStore описывает структуру хранилища БД.
type URLPgStore struct {
	pool     PgxPoolI
	urlDelCh chan urlDelBatchData

	ctx       context.Context
	ctxCancel context.CancelFunc
	wg        sync.WaitGroup
}

// urlDelBatchData описывает структуру данных для массового удаления ссылок пользователя.
type urlDelBatchData struct {
	urls   []string
	userID int
}

const (
	// Максимальное количество ссылок в очереди на удаление. При достижении будет выполнен запрос на удаление из БД.
	delURLsBatchSize = 100
	// Интервал между запуском удаления ссылок из БД (даже если не было достигнуто количество delURLsBatchSize).
	delURLBatchInterval = 10
)

// NewURLPgStore создает новое хранилище типа БД (postgresql).
func NewURLPgStore(cfg PgConfig) (*URLPgStore, error) {
	var err error
	store := &URLPgStore{
		urlDelCh: make(chan urlDelBatchData, delURLsBatchSize),
		wg:       sync.WaitGroup{},
	}
	store.ctx, store.ctxCancel = context.WithCancel(context.Background())
	store.pool, err = initPool(store.ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize a db connection: %w", err)
	}
	if err = initSchema(store.ctx, store.pool); err != nil {
		return nil, fmt.Errorf("failed to initialize a db scheme: %w", err)
	}

	store.wg.Add(1)
	go store.flushDelURLs()

	return store, nil
}

// initPool инициализация пула для соединения с БД.
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

// initSchema выполняет миграцию (создание таблиц).
func initSchema(ctx context.Context, pool PgxPoolI) error {
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

// flushDelURLs go рутина, которая собирает ссылки на удаления и запускает функцию удаления из БД.
// Удаление происходит либо когда количество ссылок на удаление превышает delURLsBatchSize.
// Либо, даже если ссылок меньше delURLsBatchSize - каждые delURLBatchInterval секунд.
func (db *URLPgStore) flushDelURLs() {
	ticker := time.NewTicker(delURLBatchInterval * time.Second)
	defer ticker.Stop()
	defer db.wg.Done()

	var batch []urlDelBatchData

	for {
		select {
		case delURLs, ok := <-db.urlDelCh:
			if !ok {
				if len(batch) > 0 {
					db.executeDelBatch(db.ctx, batch)
					return
				}
			}
			batch = append(batch, delURLs)
			if len(batch) >= delURLsBatchSize {
				db.executeDelBatch(db.ctx, batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				db.executeDelBatch(db.ctx, batch)
				batch = batch[:0]
			}
		case <-db.ctx.Done():
			if len(batch) > 0 {
				logger.Log.Debug("Executing deletion while closing pg store...")
				db.executeDelBatch(db.ctx, batch)
			}
			return
		}
	}
}

// executeDelBatch реализует удаление из БД один батч запросом.
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

// SaveURL сохраняет сокращенную ссылку.
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

// SaveBatchURL сохраняет массив сокращенных ссылок.
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

// GetURL возвращает полную ссылку по сокращенной.
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

// CreateUser сохраняет нового пользователя.
func (db *URLPgStore) CreateUser(ctx context.Context) (*User, error) {
	row := db.pool.QueryRow(ctx, "INSERT INTO users DEFAULT VALUES RETURNING id")
	var user User
	if err := row.Scan(&user.ID); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return &user, nil
}

// GetUser возвращает данные пользователя по id.
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

// GetUserURLs возвращает все сохраненные ссылки пользователя.
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

// DeleteUserURLs удаляет сокращенные ссылки пользователя.
func (db *URLPgStore) DeleteUserURLs(userID int, urls []string) error {
	db.urlDelCh <- urlDelBatchData{userID: userID, urls: urls}
	return nil
}

// IsValidID проверяет валидность сокращенной ссылки (проверка формата).
func (db *URLPgStore) IsValidID(id string) bool {
	regStr := fmt.Sprintf(`^[a-zA-Z0-9]{%d}$`, urlIDLength)
	validIDReg := regexp.MustCompile(regStr)
	return validIDReg.MatchString(id)
}

// GetURLsCount возвращает количество сокращенных ссылок в БД.
func (db *URLPgStore) GetURLsCount(ctx context.Context) (int, error) {
	row := db.pool.QueryRow(ctx,
		`SELECT count(*) FROM shorten_urls WHERE is_deleted = false;`,
	)
	var count int
	if err := row.Scan(&count); err != nil {
		return count, fmt.Errorf("failed to select urls count from db: %w", err)
	}
	return count, nil
}

// GetURLsCount возвращает количество пользователей в БД.
func (db *URLPgStore) GetUsersCount(ctx context.Context) (int, error) {
	row := db.pool.QueryRow(ctx,
		`SELECT count(*) FROM users;`,
	)
	var count int
	if err := row.Scan(&count); err != nil {
		return count, fmt.Errorf("failed to select users count from db: %w", err)
	}
	return count, nil
}

// Ping проверяет связь с БД.
func (db *URLPgStore) Ping(ctx context.Context) error {
	err := db.pool.Ping(ctx)
	return err
}

// Close закрывает пулл и все соединения с БД.
func (db *URLPgStore) Close() error {
	logger.Log.Debug("Closing pg store...")
	db.ctxCancel()
	close(db.urlDelCh)
	db.wg.Wait()
	db.pool.Close()
	return nil
}
