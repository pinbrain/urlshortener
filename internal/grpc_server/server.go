// Package grpcserver содержит реализацию gRPC сервера.
package grpcserver

import (
	"context"
	"errors"
	"net"

	"github.com/pinbrain/urlshortener/internal/grpc_server/interceptors"
	pb "github.com/pinbrain/urlshortener/internal/grpc_server/proto"
	"github.com/pinbrain/urlshortener/internal/logger"
	"github.com/pinbrain/urlshortener/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// URLShortenerServer описывает структуру gRPC сервера.
type URLShortenerServer struct {
	pb.UnimplementedURLShortenerServer
	service *service.Service
}

// NewGRPCServer создает и возвращает новый gRPC сервер.
func NewGRPCServer(service *service.Service, trustedSubnet *net.IPNet) *grpc.Server {
	ipGuardInterceptor := interceptors.NewIPGuardInterceptor(trustedSubnet)
	authInterceptor := interceptors.NewAuthInterceptor(service)
	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.LoggerInterceptor,
			authInterceptor.AuthenticateUser,
			authInterceptor.RequireUser,
			ipGuardInterceptor.GuardByIP,
		),
	)
	pb.RegisterURLShortenerServer(s, &URLShortenerServer{
		service: service,
	})

	return s
}

// ShortenURL обрабатывает запрос на сокращение ссылки.
func (s *URLShortenerServer) ShortenURL(
	ctx context.Context, in *pb.ShortenURLReq,
) (*pb.ShortenURLRes, error) {
	var response pb.ShortenURLRes
	shortenURL, err := s.service.ShortenURL(ctx, in.GetOriginalUrl())
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidURL):
			return nil, status.Error(codes.InvalidArgument, "Некорректная ссылка для сокращения")
		case errors.Is(err, service.ErrURLConflict):
			return nil, status.Error(codes.AlreadyExists, "Ссылка уже сохранена")
		default:
			logger.Log.Errorw("Error while saving url for shorten", "err", err)
			return nil, status.Error(codes.Internal, "Internal server error")
		}
	}
	response.ShortUrl = shortenURL
	return &response, nil
}

// ShortenBatchURL обрабатывает запрос на сокращение нескольких ссылок.
func (s *URLShortenerServer) ShortenBatchURL(
	ctx context.Context, in *pb.ShortenBatchURLReq,
) (*pb.ShortenBatchURLRes, error) {
	response := pb.ShortenBatchURLRes{
		Urls: []*pb.ShortenBatchURLRes_BatchURL{},
	}
	var batchURL []service.BatchURL
	for _, url := range in.GetUrls() {
		batchURL = append(batchURL, service.BatchURL{
			CorrelationID: url.GetCorrelationId(),
			OriginalURL:   url.GetOriginalUrl(),
		})
	}
	savedBatch, err := s.service.ShortenBatchURL(ctx, batchURL)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrURLConflict):
			return nil, status.Error(codes.AlreadyExists, "В запросе ссылки, которые уже были ранее сохранены")
		case errors.Is(err, service.ErrNoData):
			return nil, status.Error(codes.NotFound, "Отсутствуют данные для сокращения")
		default:
			logger.Log.Errorw("Error in saving batch of urls in store", "err", err)
			return nil, status.Error(codes.Internal, "Internal server error")
		}
	}
	for _, url := range savedBatch {
		response.Urls = append(response.Urls, &pb.ShortenBatchURLRes_BatchURL{
			CorrelationId: url.CorrelationID,
			ShortUrl:      url.ShortURL,
		})
	}
	return &response, nil
}

// GetURL обрабатывает запрос на получение полной ссылки по сокращенному id.
func (s *URLShortenerServer) GetURL(
	ctx context.Context, in *pb.GetURLReq,
) (*pb.GetURLRes, error) {
	var response pb.GetURLRes
	url, err := s.service.GetURL(ctx, in.GetUrlId())
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidURL):
			return nil, status.Error(codes.InvalidArgument, "Некорректная ссылка")
		case errors.Is(err, service.ErrIsDeleted):
			return nil, status.Error(codes.NotFound, "Ссылка удалена")
		case errors.Is(err, service.ErrNotFound):
			return nil, status.Error(codes.NotFound, "Ссылка не найдена")
		default:
			logger.Log.Errorw("Error getting original url", "err", err)
			return nil, status.Error(codes.Internal, "Internal server error")
		}
	}
	response.OriginalUrl = url
	return &response, nil
}

// GetUserURLs обрабатывает запрос на получение ссылок, сокращенных пользователем.
func (s *URLShortenerServer) GetUserURLs(
	ctx context.Context, _ *pb.GetUsersURLsReq,
) (*pb.GetUsersURLsRes, error) {
	response := pb.GetUsersURLsRes{
		Urls: []*pb.GetUsersURLsRes_UserURL{},
	}
	userURLs, err := s.service.GetUserURLs(ctx)
	if err != nil {
		logger.Log.Errorw("Error getting user urls", "err", err)
		return nil, status.Error(codes.Internal, "Internal server error")
	}
	for _, url := range userURLs {
		response.Urls = append(response.Urls, &pb.GetUsersURLsRes_UserURL{
			OriginalUrl: url.OriginalURL,
			ShortUrl:    url.ShortURL,
		})
	}
	return &response, nil
}

// DeleteUserURLs обрабатывает запрос на удаление сокращенных ссылок.
func (s *URLShortenerServer) DeleteUserURLs(
	ctx context.Context, in *pb.DeleteUserURLsReq,
) (*pb.DeleteUserURLsRes, error) {
	var response pb.DeleteUserURLsRes

	err := s.service.DeleteUserURLs(ctx, in.GetUrls())
	if err != nil {
		if errors.Is(err, service.ErrNoData) {
			return nil, status.Error(codes.NotFound, "Отсутствуют данные для удаления")
		}
		logger.Log.Errorw("Error deleting user urls", "err", err)
		return nil, status.Error(codes.Internal, "Internal server error")
	}

	return &response, nil
}

// GetStats обрабатывает запрос на получение статистики хранилища.
func (s *URLShortenerServer) GetStats(
	ctx context.Context, _ *pb.GetStatsReq,
) (*pb.GetStatsRes, error) {
	var response pb.GetStatsRes
	urls, err := s.service.GetURLsCount(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "Internal server error")
	}
	users, err := s.service.GetUsersCount(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "Internal server error")
	}
	response.Urls = int32(urls)
	response.Users = int32(users)
	return &response, nil
}

// Ping обрабатывает запрос на проверку соединения с хранилищем данных.
func (s *URLShortenerServer) Ping(_ context.Context, _ *pb.PingReq) (*pb.PingRes, error) {
	var response pb.PingRes
	return &response, nil
}
