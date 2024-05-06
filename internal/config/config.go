package config

import (
	"flag"
	"net/url"

	"github.com/caarlos0/env/v11"
)

type ServerConf struct {
	ServerAddress string  `env:"SERVER_ADDRESS"`
	BaseURL       url.URL `env:"BASE_URL"`
}

func loadFlags(cfg *ServerConf) error {
	flag.StringVar(&cfg.ServerAddress, "a", ":8080", "Адрес запуска HTTP-сервера")
	baseURLStr := flag.String("b", "http://localhost:8080", "Базовый адрес результирующего сокращённого URL")
	flag.Parse()

	if *baseURLStr != "" {
		parsedURL, err := url.ParseRequestURI(*baseURLStr)
		if err != nil {
			return err
		}
		cfg.BaseURL = *parsedURL
	}

	return nil
}

func loadEnvs(cfg *ServerConf) error {
	err := env.Parse(cfg)
	if err != nil {
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
