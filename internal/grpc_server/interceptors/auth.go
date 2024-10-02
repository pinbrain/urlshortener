package interceptors

import (
	"context"
	"errors"
	"strconv"

	appCtx "github.com/pinbrain/urlshortener/internal/context"
	pb "github.com/pinbrain/urlshortener/internal/grpc_server/proto"
	"github.com/pinbrain/urlshortener/internal/logger"
	"github.com/pinbrain/urlshortener/internal/service"
	"github.com/pinbrain/urlshortener/internal/storage"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Ключ с id пользователя.
const userIDMetaKey = "user_id"

// Перечень методов, доступных для авторизованных пользователей.
var authProtectedMethods = map[string]bool{
	pb.URLShortener_ShortenURL_FullMethodName:      true,
	pb.URLShortener_ShortenBatchURL_FullMethodName: true,
	pb.URLShortener_GetUserURLs_FullMethodName:     true,
	pb.URLShortener_DeleteUserURLs_FullMethodName:  true,
}

// AuthInterceptor описывает структуру перехватчика для авторизации и аутентификации.
type AuthInterceptor struct {
	service *service.Service
}

// NewAuthInterceptor создает обработчик авторизации и аутентификации.
func NewAuthInterceptor(service *service.Service) *AuthInterceptor {
	return &AuthInterceptor{
		service: service,
	}
}

// AuthenticateUser аутентифицирует пользователя запроса.
func (i *AuthInterceptor) AuthenticateUser(
	ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
) (interface{}, error) {
	var userID int
	var err error
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		values := md.Get(userIDMetaKey)
		if len(values) > 0 {
			idString := values[0]
			userID, err = strconv.Atoi(idString)
			if err != nil {
				logger.Log.Errorw("Wrong user id format", "err", err)
				return nil, status.Error(codes.Unauthenticated, "Wrong user id format")
			}
		}
	}

	var userData *storage.User

	if userID != 0 {
		userData, err = i.service.GetUser(ctx, userID)
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				return nil, status.Error(codes.Unauthenticated, "Request user not found")
			}
			logger.Log.Errorw("Error in getting user data", "err", err)
			return nil, status.Error(codes.Unauthenticated, "Failed to get request user data")
		}
	} else {
		userData, err = i.service.CreateUser(ctx)
		if err != nil {
			logger.Log.Errorw("Error creating new user", "err", err)
			return nil, status.Error(codes.Internal, "Internal Server Error")
		}
	}

	ctx = appCtx.CtxWithUser(ctx, &appCtx.CtxUser{ID: userData.ID})

	return handler(ctx, req)
}

// RequireUser проверяет что пользователь авторизован.
// В противном случае прерывает обработку запроса и возвращает ошибку Unauthorized.
func (i *AuthInterceptor) RequireUser(
	ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
) (interface{}, error) {
	if !authProtectedMethods[info.FullMethod] {
		return handler(ctx, req)
	}
	user := appCtx.GetCtxUser(ctx)
	if user == nil || user.ID <= 0 {
		return nil, status.Error(codes.Unauthenticated, "Unauthorized")
	}
	return handler(ctx, req)
}
