package storage

import (
	"context"
	"errors"
)

// Длина сокращенной ссылки.
const urlIDLength = 8

// ErrConflict - ошибка, указывающая на конфликт данных в хранилище.
var ErrConflict = errors.New("data conflict")

// ErrNoData - ошибка, указывающая на отсутствие данных в хранилище.
var ErrNoData = errors.New("no data")

// ErrIsDeleted - ошибка, указывающая на то, что ссылка была удалена.
var ErrIsDeleted = errors.New("deleted")

// ErrNotImplemented - ошибка, указывающая на то, что метод не реализован.
var ErrNotImplemented = errors.New("not implemented")

// URLStorage описывает интерфейс хранилища приложения.
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
	// Удалить сокращенные ссылки пользователя
	DeleteUserURLs(userID int, urls []string) error
	// Проверить валидность сокращенной ссылки (проверка формата)
	IsValidID(id string) bool
	// Проверка связи с БД (для всех остальных хранилищ ничего не делает)
	Ping(ctx context.Context) error
	// Получить количество ссылок в хранилище.
	GetURLsCount(ctx context.Context) (count int, err error)
	// Получить количество пользователей в хранилище.
	GetUsersCount(ctx context.Context) (count int, err error)
	// Закрыть хранилище (БД или файл)
	Close() error
}

// ShortenURL описывает структуру представляющую пару оригинальной и сокращенной ссылок.
type ShortenURL struct {
	Original string
	Shorten  string
}

// User описывает структуру данных пользователя.
type User struct {
	ID int
}

// URLStorageConfig описывает структуру конфигурации хранилища приложения.
type URLStorageConfig struct {
	StorageFile string
	DSN         string
}

// NewURLStorage создает новое хранилище согласно переданным настройкам.
func NewURLStorage(cfg URLStorageConfig) (URLStorage, error) {
	if cfg.DSN != "" {
		return NewURLPgStore(PgConfig{DSN: cfg.DSN})
	}
	return NewURLMapStore(cfg.StorageFile)
}
