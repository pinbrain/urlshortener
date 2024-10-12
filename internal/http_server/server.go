// Package httpserver содержит реализацию http сервера.
package httpserver

import (
	"context"
	"net/http"

	"github.com/pinbrain/urlshortener/internal/config"
	"github.com/pinbrain/urlshortener/internal/http_server/handlers"
	"github.com/pinbrain/urlshortener/internal/logger"
	"github.com/pinbrain/urlshortener/internal/service"
	"golang.org/x/crypto/acme/autocert"
)

// URLShortenerServer описывает структуру http сервера.
type URLShortenerServer struct {
	httpServer *http.Server
	isHTTPS    bool

	urlHandler handlers.URLHandler
}

// NewHTTPServer создает и возвращает новый http сервер.
func NewHTTPServer(service *service.Service, serverConf config.ServerConf) *URLShortenerServer {
	server := &URLShortenerServer{
		isHTTPS:    serverConf.EnableHTTPS,
		urlHandler: handlers.NewURLHandler(service, serverConf.BaseURL),
	}
	urlRouter := handlers.NewURLRouter(server.urlHandler, service, serverConf.TrustedSubnet)
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
		server.httpServer = &http.Server{
			Addr:    ":443",
			Handler: urlRouter,
			// для TLS-конфигурации используем менеджер сертификатов
			TLSConfig: manager.TLSConfig(),
		}
	} else {
		server.httpServer = &http.Server{
			Addr:    serverConf.ServerAddress,
			Handler: urlRouter,
		}
	}
	return server
}

// ListenAndServe запускает прослушивание tcp адреса (старт сервера).
func (s *URLShortenerServer) ListenAndServe() error {
	if s.isHTTPS {
		return s.httpServer.ListenAndServeTLS("", "")
	}
	return s.httpServer.ListenAndServe()
}

// Shutdown завершает работу сервера.
func (s *URLShortenerServer) Shutdown(ctx context.Context) error {
	s.urlHandler.Close()
	logger.Log.Info("Handler goroutines finished")
	return s.httpServer.Shutdown(ctx)
}
