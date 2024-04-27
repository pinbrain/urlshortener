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
	baseUrl := &url.URL{
		Scheme: serverScheme,
		Host:   serverHost + serverPort,
		Path:   urlHandlerPath,
	}

	urlStore := storage.NewUrlMapStore()
	urlHandler := handlers.NewUrlHandler(urlStore, baseUrl.String())
	mux := http.NewServeMux()
	mux.HandleFunc(urlHandlerPath, urlHandler.HandleRequest)
	return http.ListenAndServe(serverPort, mux)
}
