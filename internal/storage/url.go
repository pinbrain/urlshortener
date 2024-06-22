package storage

import (
	"context"
	"errors"
)

const urlIDLength = 8

// ErrConflict указывает на конфликт данных в хранилище.
var ErrConflict = errors.New("data conflict")

// ErrNoData указывает на отсутствие данных в хранилище.
var ErrNoData = errors.New("no data")

type URLStorage interface {
	// Сохранить сокращенную ссылку
	SaveURL(ctx context.Context, url string, userID int) (id string, err error)
	// Сохранить массив ссылок
	SaveBatchURL(ctx context.Context, urls []ShortenURL, userID int) error
	// Получить полную ссылку по сокращенной
	GetURL(ctx context.Context, id string) (url string, err error)
	// Создать нового пользователя
	CreateUser(ctx context.Context) (*User, error)
	// Получить данные пользователя по ID
	GetUser(ctx context.Context, id int) (*User, error)
	// Получить все сокращенные пользователем ссылки
	GetUserURLs(ctx context.Context, id int) (urls []ShortenURL, err error)
	// Проверить валидность сокращенной ссылки (проверка формата)
	IsValidID(id string) bool
	// Проверка связи с БД (для всех остальных хранилищ ничего не делает)
	Ping(ctx context.Context) error
	// Закрыть хранилище (БД или файл)
	Close() error
}

type ShortenURL struct {
	Original string
	Shorten  string
}

type User struct {
	ID int
}

type URLStorageConfig struct {
	StorageFile string
	DSN         string
}

func NewURLStorage(ctx context.Context, cfg URLStorageConfig) (URLStorage, error) {
	if cfg.DSN != "" {
		return NewURLPgStore(ctx, PgConfig{DSN: cfg.DSN})
	}
	return NewURLMapStore(cfg.StorageFile)
}
