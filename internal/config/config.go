package config

import (
	"errors"
	"flag"
	"net/url"
	"path/filepath"

	"github.com/caarlos0/env/v11"
)

type ServerConf struct {
	ServerAddress string  `env:"SERVER_ADDRESS"`
	BaseURL       url.URL `env:"BASE_URL"`
	LogLevel      string  `env:"LOG_LEVEL"`
	StorageFile   string  `env:"FILE_STORAGE_PATH"`
	DSN           string  `env:"DATABASE_DSN"`
}

func validateBaseURL(baseURL string) (*url.URL, error) {
	parsedURL, err := url.ParseRequestURI(baseURL)
	if err != nil {
		return nil, err
	}
	return parsedURL, nil
}

func validateStorageFileName(file string) error {
	if file == "" {
		return nil
	}
	if !filepath.IsAbs(file) && !filepath.IsLocal(file) {
		return errors.New("невалидное полное имя файла с данными")
	}
	return nil
}

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
