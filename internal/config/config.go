// Модуль config формирует и проверяет конфигурацию приложения.
package config

import (
	"errors"
	"flag"
	"net/url"
	"path/filepath"

	"github.com/caarlos0/env/v11"
)

// ServerConf определяет структуру конфигурации.
type ServerConf struct {
	ServerAddress string  `env:"SERVER_ADDRESS"`    // Адрес запуска HTTP-сервера.
	BaseURL       url.URL `env:"BASE_URL"`          // Базовый адрес результирующего сокращённого URL.
	LogLevel      string  `env:"LOG_LEVEL"`         // Уровень логирования.
	StorageFile   string  `env:"FILE_STORAGE_PATH"` // Полное имя файла, куда сохраняются данные.
	DSN           string  `env:"DATABASE_DSN"`      // Строка с адресом подключения к БД.
}

// validateBaseURL проверяет корректность базового адреса сокращенных ссылок.
func validateBaseURL(baseURL string) (*url.URL, error) {
	parsedURL, err := url.ParseRequestURI(baseURL)
	if err != nil {
		return nil, err
	}
	return parsedURL, nil
}

// validateStorageFileName проверяет корректность имени файла для хранение данных.
func validateStorageFileName(file string) error {
	if file == "" {
		return nil
	}
	if !filepath.IsAbs(file) && !filepath.IsLocal(file) {
		return errors.New("невалидное полное имя файла с данными")
	}
	return nil
}

// loadFlags загружает параметры конфигурации из флагов.
func loadFlags(cfg *ServerConf) error {
	flag.StringVar(&cfg.ServerAddress, "a", ":8080", "Адрес запуска HTTP-сервера")
	flag.StringVar(&cfg.LogLevel, "l", "info", "Уровень логирования")
	flag.StringVar(&cfg.DSN, "d", "", "Строка с адресом подключения к БД")
	storageFileStr := flag.String("f", "/tmp/short-url-db.json", "Полное имя файла, куда сохраняются данные")
	baseURLStr := flag.String("b", "http://localhost:8080", "Базовый адрес результирующего сокращённого URL")
	flag.Parse()

	if *baseURLStr != "" {
		parsedURL, err := validateBaseURL(*baseURLStr)
		if err != nil {
			return err
		}
		cfg.BaseURL = *parsedURL
	}

	if err := validateStorageFileName(*storageFileStr); err != nil {
		return err
	}
	cfg.StorageFile = *storageFileStr

	return nil
}

// loadEnvs загружает параметры конфигурации из переменных окружения.
func loadEnvs(cfg *ServerConf) error {
	err := env.Parse(cfg)
	if err != nil {
		return err
	}

	if err = validateStorageFileName(cfg.StorageFile); err != nil {
		return err
	}

	return nil
}

// InitConfig формирует итоговую конфигурацию приложения.
func InitConfig() (ServerConf, error) {
	serverConf := ServerConf{}

	if err := loadFlags(&serverConf); err != nil {
		return serverConf, err
	}
	if err := loadEnvs(&serverConf); err != nil {
		return serverConf, err
	}

	return serverConf, nil
}
