package interceptors

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/golang/mock/gomock"
	appCtx "github.com/pinbrain/urlshortener/internal/context"
	pb "github.com/pinbrain/urlshortener/internal/grpc_server/proto"
	"github.com/pinbrain/urlshortener/internal/service"
	"github.com/pinbrain/urlshortener/internal/storage"
	"github.com/pinbrain/urlshortener/internal/storage/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestAuth(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockURLStorage(ctrl)
	baseURL := url.URL{Scheme: "http", Host: "localhost:8080"}
	service := service.NewService(mockStorage, baseURL)
	authInterceptor := NewAuthInterceptor(&service)
	handler := func(_ context.Context, req any) (any, error) {
		return req, nil
	}

	type urlStore struct {
		urlStoreError error
		user          *storage.User
		create        bool
	}

	tests := []struct {
		name     string
		urlStore *urlStore
		method   string
		meta     map[string]string
		wantErr  bool
		errCode  codes.Code
	}{
		{
			name: "Успешный запрос",
			urlStore: &urlStore{
				user: &storage.User{ID: 1},
			},
			method:  pb.URLShortener_ShortenURL_FullMethodName,
			meta:    map[string]string{userIDMetaKey: "1"},
			wantErr: false,
		},
		{
			name:    "Некорректный формат id",
			method:  pb.URLShortener_ShortenURL_FullMethodName,
			meta:    map[string]string{userIDMetaKey: "a"},
			wantErr: true,
			errCode: codes.Unauthenticated,
		},
		{
			name: "Пользователь не найден",
			urlStore: &urlStore{
				user:          &storage.User{ID: 1},
				urlStoreError: storage.ErrNoData,
			},
			method:  pb.URLShortener_ShortenURL_FullMethodName,
			meta:    map[string]string{userIDMetaKey: "1"},
			wantErr: true,
			errCode: codes.Unauthenticated,
		},
		{
			name:    "id < 0",
			method:  pb.URLShortener_ShortenURL_FullMethodName,
			meta:    map[string]string{userIDMetaKey: "-1"},
			wantErr: true,
			errCode: codes.Unauthenticated,
		},
		{
			name: "Создание нового пользователя",
			urlStore: &urlStore{
				user:   &storage.User{ID: 1},
				create: true,
			},
			method:  pb.URLShortener_ShortenURL_FullMethodName,
			wantErr: false,
		},
		{
			name: "Ошибка создания нового пользователя",
			urlStore: &urlStore{
				user:          &storage.User{ID: 1},
				create:        true,
				urlStoreError: errors.New("store error"),
			},
			method:  pb.URLShortener_ShortenURL_FullMethodName,
			wantErr: true,
			errCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.urlStore != nil {
				if tt.urlStore.create {
					mockStorage.EXPECT().CreateUser(gomock.Any()).
						Times(1).Return(tt.urlStore.user, tt.urlStore.urlStoreError)
				} else {
					mockStorage.EXPECT().GetUser(gomock.Any(), tt.urlStore.user.ID).
						Times(1).Return(tt.urlStore.user, tt.urlStore.urlStoreError)
				}
			} else {
				mockStorage.EXPECT().GetUser(gomock.Any(), gomock.Any()).Times(0)
				mockStorage.EXPECT().CreateUser(gomock.Any()).Times(0)
			}

			md := metadata.New(tt.meta)
			ctx := metadata.NewIncomingContext(context.Background(), md)
			info := &grpc.UnaryServerInfo{FullMethod: tt.method}
			_, err := authInterceptor.AuthenticateUser(ctx, nil, info, handler)
			if !tt.wantErr {
				require.NoError(t, err)
			} else {
				code, _ := status.FromError(err)
				assert.Equal(t, tt.errCode, code.Code())
			}
		})
	}
}

func TestRequireUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockURLStorage(ctrl)
	baseURL := url.URL{Scheme: "http", Host: "localhost:8080"}
	service := service.NewService(mockStorage, baseURL)
	authInterceptor := NewAuthInterceptor(&service)
	handler := func(_ context.Context, req any) (any, error) {
		return req, nil
	}

	tests := []struct {
		name    string
		method  string
		user    *appCtx.CtxUser
		wantErr bool
		errCode codes.Code
	}{
		{
			name:    "Успешный запрос в защищенный метод",
			method:  pb.URLShortener_ShortenURL_FullMethodName,
			user:    &appCtx.CtxUser{ID: 1},
			wantErr: false,
		},
		{
			name:    "Успешный запрос в незащищенный метод",
			method:  pb.URLShortener_GetStats_FullMethodName,
			wantErr: false,
		},
		{
			name:    "Ошибка",
			method:  pb.URLShortener_ShortenURL_FullMethodName,
			wantErr: true,
			errCode: codes.Unauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.user != nil {
				ctx = appCtx.CtxWithUser(ctx, tt.user)
			}
			info := &grpc.UnaryServerInfo{FullMethod: tt.method}
			_, err := authInterceptor.RequireUser(ctx, nil, info, handler)
			if !tt.wantErr {
				require.NoError(t, err)
			} else {
				code, _ := status.FromError(err)
				assert.Equal(t, tt.errCode, code.Code())
			}
		})
	}
}
