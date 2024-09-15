// Package app инициализирует и запускает сервис сокращения ссылок.
package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/pinbrain/urlshortener/internal/config"
	"github.com/pinbrain/urlshortener/internal/handlers"
	"github.com/pinbrain/urlshortener/internal/logger"
	"github.com/pinbrain/urlshortener/internal/storage"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/sync/errgroup"
)

const (
	timeoutServerShutdown = time.Second * 5
	timeoutShutdown       = time.Second * 10
)

// Run загружает конфигурацию, создает хранилище согласно настройкам, запускает http сервер приложения.
func Run() error {
	rootCtx, cancelCtx := signal.NotifyContext(
		context.Background(),
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGQUIT,
	)
	defer cancelCtx()

	g, ctx := errgroup.WithContext(rootCtx)
	// нештатное завершение программы по таймауту
	// происходит, если после завершения контекста
	// приложение не смогло завершиться за отведенный промежуток времени
	context.AfterFunc(ctx, func() {
		afterCtx, afterCancelCtx := context.WithTimeout(context.Background(), timeoutShutdown)
		defer afterCancelCtx()

		<-afterCtx.Done()
		log.Fatal("failed to gracefully shutdown the service")
	})

	serverConf, err := config.InitConfig()
	if err != nil {
		return err
	}

	if err = logger.Initialize(serverConf.LogLevel); err != nil {
		return err
	}

	urlStore, err := storage.NewURLStorage(storage.URLStorageConfig{
		StorageFile: serverConf.StorageFile,
		DSN:         serverConf.DSN,
	})
	if err != nil {
		return err
	}

	urlHandler := handlers.NewURLHandler(urlStore, serverConf.BaseURL)

	logger.Log.Infow("Starting server", "addr", serverConf.ServerAddress)

	var server *http.Server

	// Запуск сервера
	g.Go(func() (err error) {
		defer func() {
			errRec := recover()
			if errRec != nil {
				err = fmt.Errorf("a panic occurred: %v", errRec)
			}
		}()
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
			server = &http.Server{
				Addr:    ":443",
				Handler: handlers.NewURLRouter(urlHandler, urlStore),
				// для TLS-конфигурации используем менеджер сертификатов
				TLSConfig: manager.TLSConfig(),
			}
			if err = server.ListenAndServeTLS("", ""); err != nil {
				if errors.Is(err, http.ErrServerClosed) {
					return nil
				}
				return fmt.Errorf("listen and server has failed: %w", err)
			}
			return nil
		}
		server = &http.Server{
			Addr:    serverConf.ServerAddress,
			Handler: handlers.NewURLRouter(urlHandler, urlStore),
		}
		if err = server.ListenAndServe(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				return nil
			}
			return fmt.Errorf("listen and server has failed: %w", err)
		}
		return nil
	})

	// Отслеживаем успешное завершение работы сервера
	g.Go(func() error {
		defer logger.Log.Info("Service has been shutdown")

		<-ctx.Done()
		logger.Log.Info("Gracefully shutting down service...")

		shutdownTimeoutCtx, cancelShutdownTimeoutCtx := context.WithTimeout(context.Background(), timeoutServerShutdown)
		defer cancelShutdownTimeoutCtx()

		if err = server.Shutdown(shutdownTimeoutCtx); err != nil {
			logger.Log.Errorf("an error occurred during server shutdown: %v", err)
		}
		logger.Log.Info("HTTP server stopped")

		urlStore.Close()
		logger.Log.Info("URL store closed")

		return nil
	})

	if err = g.Wait(); err != nil {
		return err
	}

	return nil
}
