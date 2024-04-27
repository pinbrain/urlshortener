package app

import (
	"net/http"
	"net/url"

	"github.com/pinbrain/urlshortener/internal/handlers"
	"github.com/pinbrain/urlshortener/internal/storage"
)

const (
	serverScheme   = "http"
	serverHost     = "localhost"
	serverPort     = ":8080"
	urlHandlerPath = "/"
)

func Run() error {
	baseURL := &url.URL{
		Scheme: serverScheme,
		Host:   serverHost + serverPort,
		Path:   urlHandlerPath,
	}

	urlStore := storage.NewURLMapStore()
	urlHandler := handlers.NewURLHandler(urlStore, baseURL.String())
	mux := http.NewServeMux()
	mux.HandleFunc(urlHandlerPath, urlHandler.HandleRequest)
	return http.ListenAndServe(serverPort, mux)
}
