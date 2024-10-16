// Package app инициализирует и запускает сервис сокращения ссылок.
package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/pinbrain/urlshortener/internal/config"
	grpcserver "github.com/pinbrain/urlshortener/internal/grpc_server"
	httpserver "github.com/pinbrain/urlshortener/internal/http_server"
	"github.com/pinbrain/urlshortener/internal/logger"
	"github.com/pinbrain/urlshortener/internal/service"
	"github.com/pinbrain/urlshortener/internal/storage"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
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

	logger.Log.Infow("Starting server", "addr", serverConf.ServerAddress)

	service := service.NewService(urlStore, serverConf.BaseURL)

	var server *httpserver.URLShortenerServer
	var grpcServer *grpc.Server

	// Запуск HTTP сервера
	g.Go(func() (err error) {
		defer func() {
			errRec := recover()
			if errRec != nil {
				err = fmt.Errorf("a panic occurred: %v", errRec)
			}
		}()
		server = httpserver.NewHTTPServer(&service, serverConf)
		if err = server.ListenAndServe(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				return nil
			}
			return fmt.Errorf("listen and server has failed: %w", err)
		}
		return nil
	})

	// Запуск gRPC сервера
	g.Go(func() (err error) {
		defer func() {
			errRec := recover()
			if errRec != nil {
				err = fmt.Errorf("a panic occurred: %v", errRec)
			}
		}()
		listen, err := net.Listen("tcp", serverConf.GRPCAddress)
		if err != nil {
			return fmt.Errorf("listen tcp has failed: %w", err)
		}
		grpcServer = grpcserver.NewGRPCServer(&service, serverConf.TrustedSubnet)
		logger.Log.Infow("Starting gRPC server", "addr", serverConf.GRPCAddress)
		if err = grpcServer.Serve(listen); err != nil {
			return fmt.Errorf("listen and serve grpc has failed: %w", err)
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

		grpcServer.GracefulStop()
		logger.Log.Info("gRPC server stopped")

		urlStore.Close()
		logger.Log.Info("URL store closed")

		return nil
	})

	if err = g.Wait(); err != nil {
		return err
	}

	return nil
}
