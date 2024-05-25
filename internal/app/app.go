package app

import (
	"net/http"
	"os"

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
		r.Get("/{urlID}", urlHandler.HandleRedirect)
		r.Post("/", urlHandler.HandleShortenURL)
	})
	r.Route("/api", func(r chi.Router) {
		r.Post("/shorten", urlHandler.HandleJSONShortenURL)
	})

	return r
}

func Run() error {
	serverConf, err := config.InitConfig()
	if err != nil {
		return err
	}

	if err = logger.Initialize(serverConf.LogLevel); err != nil {
		return err
	}

	var jsonDBFile *os.File
	if serverConf.StorageFile != "" {
		jsonDBFile, err = os.OpenFile(serverConf.StorageFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			return err
		}
		defer jsonDBFile.Close()
	}
	urlStore, err := storage.NewURLMapStore(jsonDBFile)
	if err != nil {
		return err
	}

	urlHandler := handlers.NewURLHandler(urlStore, serverConf.BaseURL)

	logger.Log.Infow("Starting server", "addr", serverConf.ServerAddress)
	return http.ListenAndServe(serverConf.ServerAddress, urlRouter(urlHandler))
}
