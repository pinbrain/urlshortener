// Package config формирует и проверяет конфигурацию приложения.
package config

import (
	"encoding/json"
	"errors"
	"flag"
	"net/url"
	"os"
	"path/filepath"

	"github.com/caarlos0/env/v11"
)

// ServerConf определяет структуру конфигурации.
type ServerConf struct {
	ServerAddress string  `env:"SERVER_ADDRESS" json:"server_address"`       // Адрес запуска HTTP-сервера.
	BaseURL       url.URL `env:"BASE_URL" json:"-"`                          // Базовый адрес сокращённого URL.
	LogLevel      string  `env:"LOG_LEVEL" json:"-"`                         // Уровень логирования.
	StorageFile   string  `env:"FILE_STORAGE_PATH" json:"file_storage_path"` // Полное имя файла, куда сохраняются данные.
	DSN           string  `env:"DATABASE_DSN" json:"database_dsn"`           // Строка с адресом подключения к БД.
	EnableHTTPS   bool    `env:"ENABLE_HTTPS" json:"enable_https"`           // Признак включения HTTPS
	JSONConfig    string  `env:"CONFIG" json:"-"`                            // Имя файла json с конфигурацией
}

type JSONServerConf struct {
	ServerConf
	BaseURL string `json:"base_url"`
}

// validateBaseURL проверяет корректность базового адреса сокращенных ссылок.
func validateBaseURL(baseURL string) (*url.URL, error) {
	parsedURL, err := url.ParseRequestURI(baseURL)
	if err != nil {
		return nil, err
	}
	return parsedURL, nil
}

// validateFileName проверяет корректность имени файла для хранение данных.
func validateFileName(file string) error {
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
	flag.BoolVar(&cfg.EnableHTTPS, "s", false, "Флаг включения HTTPS")
	storageFileStr := flag.String("f", "/tmp/short-url-db.json", "Полное имя файла, куда сохраняются данные")
	baseURLStr := flag.String("b", "http://localhost:8080", "Базовый адрес результирующего сокращённого URL")
	configFileStr := flag.String("c", "", "Имя файла json с конфигурацией приложения")
	flag.Parse()

	if *baseURLStr != "" {
		parsedURL, err := validateBaseURL(*baseURLStr)
		if err != nil {
			return err
		}
		cfg.BaseURL = *parsedURL
	}

	if err := validateFileName(*storageFileStr); err != nil {
		return err
	}
	cfg.StorageFile = *storageFileStr

	if err := validateFileName(*configFileStr); err != nil {
		return err
	}
	cfg.JSONConfig = *configFileStr

	return nil
}

// loadEnvs загружает параметры конфигурации из переменных окружения.
func loadEnvs(cfg *ServerConf) error {
	err := env.Parse(cfg)
	if err != nil {
		return err
	}

	if err = validateFileName(cfg.StorageFile); err != nil {
		return err
	}

	if err = validateFileName(cfg.JSONConfig); err != nil {
		return err
	}

	return nil
}

// loadJSON загружает параметры конфигурации из файла json.
func loadJSON(cfg *ServerConf) error {
	if cfg.JSONConfig == "" {
		return nil
	}

	data, err := os.ReadFile(cfg.JSONConfig)
	if err != nil {
		return err
	}
	jsonCfg := &JSONServerConf{}
	if err = json.Unmarshal(data, jsonCfg); err != nil {
		return err
	}

	if cfg.BaseURL.String() == "" && jsonCfg.BaseURL != "" {
		parsedURL, err := validateBaseURL(jsonCfg.BaseURL)
		if err != nil {
			return err
		}
		cfg.BaseURL = *parsedURL
	}
	if cfg.DSN == "" {
		cfg.DSN = jsonCfg.DSN
	}
	if cfg.ServerAddress == "" {
		cfg.ServerAddress = jsonCfg.ServerAddress
	}
	if cfg.StorageFile == "" {
		cfg.StorageFile = jsonCfg.StorageFile
	}
	if !cfg.EnableHTTPS {
		cfg.EnableHTTPS = jsonCfg.EnableHTTPS
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
	if err := loadJSON(&serverConf); err != nil {
		return serverConf, err
	}

	return serverConf, nil
}
