package storage

import (
	"context"
)

const urlIDLength = 8

type URLStorage interface {
	// Сохранить сокращенную ссылку
	SaveURL(ctx context.Context, url string) (id string, err error)
	// Сохранить массив ссылок
	SaveBatchURL(ctx context.Context, urls []ShortenURL) error
	// Получить полную ссылку по сокращенной
	GetURL(ctx context.Context, id string) (url string, err error)
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
