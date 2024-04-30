package app

import (
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/pinbrain/urlshortener/internal/handlers"
	"github.com/pinbrain/urlshortener/internal/storage"
)

const (
	serverScheme   = "http"
	serverHost     = "localhost"
	serverPort     = ":8080"
	urlHandlerPath = "/"
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
	baseURL := &url.URL{
		Scheme: serverScheme,
		Host:   serverHost + serverPort,
		Path:   urlHandlerPath,
	}

	urlStore := storage.NewURLMapStore()
	urlHandler := handlers.NewURLHandler(urlStore, baseURL.String())

	return http.ListenAndServe(serverPort, urlRouter(urlHandler))
}
