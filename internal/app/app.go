package app

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pinbrain/urlshortener/internal/config"
	"github.com/pinbrain/urlshortener/internal/handlers"
	"github.com/pinbrain/urlshortener/internal/storage"
)

func urlRouter(urlHandler handlers.URLHandler) chi.Router {
	r := chi.NewRouter()

	r.Route("/", func(r chi.Router) {
		r.Get("/{urlID}", urlHandler.HandleRedirect)
		r.Post("/", urlHandler.HandleShortenURL)
	})

	return r
}

func Run() error {
	serverConf, err := config.InitConfig()
	if err != nil {
		return err
	}

	urlStore := storage.NewURLMapStore()
	urlHandler := handlers.NewURLHandler(urlStore, serverConf.BaseURL)

	fmt.Println("Running server on", serverConf.RunAddress)
	return http.ListenAndServe(serverConf.RunAddress, urlRouter(urlHandler))
}
