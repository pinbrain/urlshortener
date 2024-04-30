package config

import (
	"errors"

	"github.com/pinbrain/urlshortener/internal/utils"
)

type ServerConf struct {
	RunAddress string
	BaseURL    string
}

func InitConfig() (ServerConf, error) {
	serverConf := ServerConf{}
	parseFlags(&serverConf)
	if isValidBaseURL := utils.IsValidURLString(serverConf.BaseURL); !isValidBaseURL {
		return serverConf, errors.New("невалидный адрес результирующего сокращённого URL")
	}
	return serverConf, nil
}
