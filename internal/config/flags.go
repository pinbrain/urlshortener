package config

import "flag"

func parseFlags(serverConf *ServerConf) {
	flag.StringVar(&serverConf.RunAddress, "a", ":8080", "Адрес запуска HTTP-сервера")
	flag.StringVar(&serverConf.BaseURL, "b", "http://localhost:8080/", "Базовый адрес результирующего сокращённого URL")
	flag.Parse()
}
