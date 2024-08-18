package app

import (
	"context"
	"net/http"

	"github.com/pinbrain/urlshortener/internal/config"
	"github.com/pinbrain/urlshortener/internal/handlers"
	"github.com/pinbrain/urlshortener/internal/logger"
	"github.com/pinbrain/urlshortener/internal/storage"
)

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
	return http.ListenAndServe(serverConf.ServerAddress, handlers.NewURLRouter(urlHandler, urlStore))
}
