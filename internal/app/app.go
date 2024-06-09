package app

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pinbrain/urlshortener/internal/config"
	"github.com/pinbrain/urlshortener/internal/handlers"
	"github.com/pinbrain/urlshortener/internal/logger"
	"github.com/pinbrain/urlshortener/internal/middleware"
	"github.com/pinbrain/urlshortener/internal/storage"
)

func urlRouter(urlHandler handlers.URLHandler) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.HTTPRequestLogger)
	r.Use(middleware.GzipMiddleware)

	r.Route("/", func(r chi.Router) {
		r.Get("/ping", urlHandler.HandlePing)
		r.Get("/{urlID}", urlHandler.HandleRedirect)
		r.Post("/", urlHandler.HandleShortenURL)
	})
	r.Route("/api", func(r chi.Router) {
		r.Post("/shorten", urlHandler.HandleJSONShortenURL)
		r.Post("/shorten/batch", urlHandler.HandleShortenBatchURL)
	})

	return r
}

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
	return http.ListenAndServe(serverConf.ServerAddress, urlRouter(urlHandler))
}
