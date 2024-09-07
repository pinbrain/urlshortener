// Package app инициализирует и запускает сервис сокращения ссылок.
package app

import (
	"context"
	"net/http"

	"github.com/pinbrain/urlshortener/internal/config"
	"github.com/pinbrain/urlshortener/internal/handlers"
	"github.com/pinbrain/urlshortener/internal/logger"
	"github.com/pinbrain/urlshortener/internal/storage"
	"golang.org/x/crypto/acme/autocert"
)

// Run загружает конфигурацию, создает хранилище согласно настройкам, запускает http сервер приложения.
func Run() error {
	ctx := context.Background()
	serverConf, err := config.InitConfig()
	if err != nil {
		return err
	}

	if err = logger.Initialize(serverConf.LogLevel); err != nil {
		return err
	}

	urlStore, err := storage.NewURLStorage(ctx, storage.URLStorageConfig{
		StorageFile: serverConf.StorageFile,
		DSN:         serverConf.DSN,
	})
	if err != nil {
		return err
	}
	defer urlStore.Close()

	urlHandler := handlers.NewURLHandler(urlStore, serverConf.BaseURL)

	logger.Log.Infow("Starting server", "addr", serverConf.ServerAddress)

	if serverConf.EnableHTTPS {
		manager := &autocert.Manager{
			// директория для хранения сертификатов
			Cache: autocert.DirCache("cache-dir"),
			// функция, принимающая Terms of Service издателя сертификатов
			Prompt: autocert.AcceptTOS,
			// перечень доменов, для которых будут поддерживаться сертификаты
			HostPolicy: autocert.HostWhitelist("mysite.ru"),
		}
		// конструируем сервер с поддержкой TLS
		server := &http.Server{
			Addr:    ":443",
			Handler: handlers.NewURLRouter(urlHandler, urlStore),
			// для TLS-конфигурации используем менеджер сертификатов
			TLSConfig: manager.TLSConfig(),
		}
		return server.ListenAndServeTLS("", "")
	}
	return http.ListenAndServe(serverConf.ServerAddress, handlers.NewURLRouter(urlHandler, urlStore))
}
