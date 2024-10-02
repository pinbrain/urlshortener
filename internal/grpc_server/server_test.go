package grpcserver

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
	"google.golang.org/grpc/status"
)

func TestNewGRPCServer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockURLStorage(ctrl)
	baseURL := url.URL{Scheme: "http", Host: "localhost:8080"}
	service := service.NewService(mockStorage, baseURL)
	server := NewGRPCServer(&service, nil)
	assert.IsType(t, (*grpc.Server)(nil), server)
}

func TestShortenURL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockURLStorage(ctrl)
	baseURL := url.URL{Scheme: "http", Host: "localhost:8080"}
	service := service.NewService(mockStorage, baseURL)
	server := URLShortenerServer{service: &service}

	type urlStore struct {
		urlStoreError error
		urlID         string
	}

	tests := []struct {
		name     string
		urlStore *urlStore
		request  *pb.ShortenURLReq
		expected *pb.ShortenURLRes
		wantErr  bool
		errCode  codes.Code
	}{
		{
			name: "Успешный запрос",
			urlStore: &urlStore{
				urlID: "abc",
			},
			request:  &pb.ShortenURLReq{OriginalUrl: "http://some.ru"},
			expected: &pb.ShortenURLRes{ShortUrl: "http://localhost:8080/abc"},
			wantErr:  false,
		},
		{
			name:    "Некорректная ссылка",
			request: &pb.ShortenURLReq{OriginalUrl: "invalid url"},
			wantErr: true,
			errCode: codes.InvalidArgument,
		},
		{
			name: "Ссылка уже есть",
			urlStore: &urlStore{
				urlID:         "abc",
				urlStoreError: storage.ErrConflict,
			},
			request: &pb.ShortenURLReq{OriginalUrl: "http://some.ru"},
			wantErr: true,
			errCode: codes.AlreadyExists,
		},
		{
			name: "Ошибка хранилища",
			urlStore: &urlStore{
				urlStoreError: errors.New("store error"),
			},
			request: &pb.ShortenURLReq{OriginalUrl: "http://some.ru"},
			wantErr: true,
			errCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.urlStore != nil {
				mockStorage.EXPECT().SaveURL(gomock.Any(), tt.request.GetOriginalUrl(), gomock.Any()).
					Times(1).Return(tt.urlStore.urlID, tt.urlStore.urlStoreError)
			} else {
				mockStorage.EXPECT().SaveURL(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			}

			response, err := server.ShortenURL(context.Background(), tt.request)
			if !tt.wantErr {
				require.NoError(t, err)
				assert.Equal(t, tt.expected.GetShortUrl(), response.GetShortUrl())
			} else {
				code, _ := status.FromError(err)
				assert.Equal(t, tt.errCode, code.Code())
			}
		})
	}
}

func TestShortenBatchURL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockURLStorage(ctrl)
	baseURL := url.URL{Scheme: "http", Host: "localhost:8080"}
	service := service.NewService(mockStorage, baseURL)
	server := URLShortenerServer{service: &service}

	type urlStore struct {
		urlStoreError error
		shortURLs     []storage.ShortenURL
	}

	tests := []struct {
		name     string
		urlStore *urlStore
		request  *pb.ShortenBatchURLReq
		expected *pb.ShortenBatchURLRes
		wantErr  bool
		errCode  codes.Code
	}{
		{
			name: "Успешный запрос",
			urlStore: &urlStore{
				shortURLs: []storage.ShortenURL{
					{Shorten: "abc1"},
					{Shorten: "abc2"},
				},
			},
			request: &pb.ShortenBatchURLReq{
				Urls: []*pb.ShortenBatchURLReq_BatchURL{
					{CorrelationId: "1", OriginalUrl: "http://some1.ru"},
					{CorrelationId: "2", OriginalUrl: "http://some2.ru"},
				},
			},
			expected: &pb.ShortenBatchURLRes{
				Urls: []*pb.ShortenBatchURLRes_BatchURL{
					{CorrelationId: "1", ShortUrl: "http://localhost:8080/abc1"},
					{CorrelationId: "2", ShortUrl: "http://localhost:8080/abc2"},
				},
			},
			wantErr: false,
		},
		{
			name: "Ссылка уже есть",
			urlStore: &urlStore{
				urlStoreError: storage.ErrConflict,
			},
			request: &pb.ShortenBatchURLReq{
				Urls: []*pb.ShortenBatchURLReq_BatchURL{
					{CorrelationId: "1", OriginalUrl: "http://some1.ru"},
					{CorrelationId: "2", OriginalUrl: "http://some2.ru"},
				},
			},
			wantErr: true,
			errCode: codes.AlreadyExists,
		},
		{
			name: "Ошибка хранилища",
			urlStore: &urlStore{
				urlStoreError: errors.New("store error"),
			},
			request: &pb.ShortenBatchURLReq{
				Urls: []*pb.ShortenBatchURLReq_BatchURL{
					{CorrelationId: "1", OriginalUrl: "http://some1.ru"},
					{CorrelationId: "2", OriginalUrl: "http://some2.ru"},
				},
			},
			wantErr: true,
			errCode: codes.Internal,
		},
		{
			name: "Нет данных",
			request: &pb.ShortenBatchURLReq{
				Urls: []*pb.ShortenBatchURLReq_BatchURL{},
			},
			wantErr: true,
			errCode: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.urlStore != nil {
				if tt.urlStore.urlStoreError != nil {
					mockStorage.EXPECT().
						SaveBatchURL(gomock.Any(), gomock.Any(), gomock.Any()).
						Times(1).
						Return(tt.urlStore.urlStoreError)
				} else {
					mockStorage.EXPECT().
						SaveBatchURL(gomock.Any(), gomock.Any(), gomock.Any()).
						DoAndReturn(func(_ context.Context, batch []storage.ShortenURL, _ int) error {
							for i := range batch {
								batch[i].Shorten = tt.urlStore.shortURLs[i].Shorten
							}
							return nil
						}).
						Times(1)
				}
			}
			response, err := server.ShortenBatchURL(context.Background(), tt.request)
			if !tt.wantErr {
				require.NoError(t, err)
				for i, val := range response.GetUrls() {
					assert.Equal(t, tt.expected.GetUrls()[i].GetCorrelationId(), val.GetCorrelationId())
					assert.Equal(t, tt.expected.GetUrls()[i].GetShortUrl(), val.GetShortUrl())
				}
			} else {
				code, _ := status.FromError(err)
				assert.Equal(t, tt.errCode, code.Code())
			}
		})
	}
}

func TestGetURL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockURLStorage(ctrl)
	baseURL := url.URL{Scheme: "http", Host: "localhost:8080"}
	service := service.NewService(mockStorage, baseURL)
	server := URLShortenerServer{service: &service}

	type urlStore struct {
		urlStoreError error
		url           string
		isValid       bool
	}

	tests := []struct {
		name     string
		urlStore *urlStore
		request  *pb.GetURLReq
		expected *pb.GetURLRes
		wantErr  bool
		errCode  codes.Code
	}{
		{
			name: "Успешный запрос",
			urlStore: &urlStore{
				url:     "http://some.ru",
				isValid: true,
			},
			request:  &pb.GetURLReq{UrlId: "abc"},
			expected: &pb.GetURLRes{OriginalUrl: "http://some.ru"},
			wantErr:  false,
		},
		{
			name: "Некорректная ссылка",
			urlStore: &urlStore{
				url:     "http://some.ru",
				isValid: false,
			},
			request: &pb.GetURLReq{UrlId: "abc"},
			wantErr: true,
			errCode: codes.InvalidArgument,
		},
		{
			name: "Ссылка удалена",
			urlStore: &urlStore{
				url:           "http://some.ru",
				isValid:       true,
				urlStoreError: storage.ErrIsDeleted,
			},
			request: &pb.GetURLReq{UrlId: "abc"},
			wantErr: true,
			errCode: codes.NotFound,
		},
		{
			name: "Ссылка не найдена",
			urlStore: &urlStore{
				url:     "",
				isValid: true,
			},
			request: &pb.GetURLReq{UrlId: "abc"},
			wantErr: true,
			errCode: codes.NotFound,
		},
		{
			name: "Ошибка хранилища",
			urlStore: &urlStore{
				urlStoreError: errors.New("store error"),
				isValid:       true,
			},
			request: &pb.GetURLReq{UrlId: "abc"},
			wantErr: true,
			errCode: codes.Internal,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.urlStore != nil {
				mockStorage.EXPECT().IsValidID(tt.request.GetUrlId()).
					Times(1).Return(tt.urlStore.isValid)
				if tt.urlStore.isValid {
					mockStorage.EXPECT().GetURL(gomock.Any(), tt.request.GetUrlId()).
						Times(1).Return(tt.urlStore.url, tt.urlStore.urlStoreError)
				}
			} else {
				mockStorage.EXPECT().IsValidID(gomock.Any()).Times(0)
				mockStorage.EXPECT().GetURL(gomock.Any(), gomock.Any()).Times(0)
			}
			response, err := server.GetURL(context.Background(), tt.request)
			if !tt.wantErr {
				require.NoError(t, err)
				assert.Equal(t, tt.expected.GetOriginalUrl(), response.GetOriginalUrl())
			} else {
				code, _ := status.FromError(err)
				assert.Equal(t, tt.errCode, code.Code())
			}
		})
	}
}

func TestGetUserURLs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockURLStorage(ctrl)
	baseURL := url.URL{Scheme: "http", Host: "localhost:8080"}
	service := service.NewService(mockStorage, baseURL)
	server := URLShortenerServer{service: &service}

	type urlStore struct {
		storeError error
		userURLs   []storage.ShortenURL
	}

	tests := []struct {
		name     string
		urlStore *urlStore
		user     *appCtx.CtxUser
		expected *pb.GetUsersURLsRes
		wantErr  bool
		errCode  codes.Code
	}{
		{
			name: "Успешный запрос",
			urlStore: &urlStore{
				userURLs: []storage.ShortenURL{
					{Original: "http://some1.ru", Shorten: "abc1"},
					{Original: "http://some2.ru", Shorten: "abc2"},
				},
			},
			user: &appCtx.CtxUser{ID: 1},
			expected: &pb.GetUsersURLsRes{Urls: []*pb.GetUsersURLsRes_UserURL{
				{OriginalUrl: "http://some1.ru", ShortUrl: "abc1"},
				{OriginalUrl: "http://some2.ru", ShortUrl: "abc2"},
			}},
			wantErr: false,
		},
		{
			name: "Ошибка хранилища",
			urlStore: &urlStore{
				userURLs: []storage.ShortenURL{
					{Original: "http://some1.ru", Shorten: "abc1"},
					{Original: "http://some2.ru", Shorten: "abc2"},
				},
				storeError: errors.New("store error"),
			},
			user:    &appCtx.CtxUser{ID: 1},
			wantErr: true,
			errCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.urlStore != nil {
				mockStorage.EXPECT().GetUserURLs(gomock.Any(), tt.user.ID).
					Times(1).Return(tt.urlStore.userURLs, tt.urlStore.storeError)
			} else {
				mockStorage.EXPECT().GetUserURLs(gomock.Any(), gomock.Any()).Times(0)
			}
			ctx := context.Background()
			if tt.user != nil {
				ctx = appCtx.CtxWithUser(ctx, tt.user)
			}
			response, err := server.GetUserURLs(ctx, &pb.GetUsersURLsReq{})
			if !tt.wantErr {
				require.NoError(t, err)
				assert.ElementsMatch(t, tt.expected.GetUrls(), response.GetUrls())
			} else {
				code, _ := status.FromError(err)
				assert.Equal(t, tt.errCode, code.Code())
			}
		})
	}
}

func TestDeleteUserURLs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockURLStorage(ctrl)
	baseURL := url.URL{Scheme: "http", Host: "localhost:8080"}
	service := service.NewService(mockStorage, baseURL)
	server := URLShortenerServer{service: &service}

	type urlStore struct {
		storeError error
		urls       []string
	}

	tests := []struct {
		name     string
		urlStore *urlStore
		user     *appCtx.CtxUser
		request  *pb.DeleteUserURLsReq
		wantErr  bool
		errCode  codes.Code
	}{
		{
			name: "Успешный запрос",
			user: &appCtx.CtxUser{ID: 1},
			urlStore: &urlStore{
				urls: []string{"abc1", "abc2"},
			},
			request: &pb.DeleteUserURLsReq{Urls: []string{"abc1", "abc2"}},
			wantErr: false,
		},
		{
			name:    "Нет данных для удаления",
			user:    &appCtx.CtxUser{ID: 1},
			request: &pb.DeleteUserURLsReq{Urls: []string{}},
			wantErr: true,
			errCode: codes.NotFound,
		},
		{
			name: "Ошибка хранилища",
			user: &appCtx.CtxUser{ID: 1},
			urlStore: &urlStore{
				storeError: errors.New("store error"),
				urls:       []string{"abc1", "abc2"},
			},
			request: &pb.DeleteUserURLsReq{Urls: []string{"abc1", "abc2"}},
			wantErr: true,
			errCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.urlStore != nil {
				mockStorage.EXPECT().DeleteUserURLs(tt.user.ID, tt.urlStore.urls).
					Times(1).Return(tt.urlStore.storeError)
			} else {
				mockStorage.EXPECT().DeleteUserURLs(gomock.Any(), gomock.Any()).Times(0)
			}
			ctx := context.Background()
			if tt.user != nil {
				ctx = appCtx.CtxWithUser(ctx, tt.user)
			}
			_, err := server.DeleteUserURLs(ctx, tt.request)
			if !tt.wantErr {
				require.NoError(t, err)
			} else {
				code, _ := status.FromError(err)
				assert.Equal(t, tt.errCode, code.Code())
			}
		})
	}
}

func TestGetStats(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockURLStorage(ctrl)
	baseURL := url.URL{Scheme: "http", Host: "localhost:8080"}
	service := service.NewService(mockStorage, baseURL)
	server := URLShortenerServer{service: &service}

	type stat struct {
		err   error
		count int
	}
	type urlStore struct {
		users *stat
		urls  *stat
	}

	tests := []struct {
		name     string
		urlStore urlStore
		expected *pb.GetStatsRes
		wantErr  bool
		errCode  codes.Code
	}{
		{
			name: "Успешный запрос",
			urlStore: urlStore{
				users: &stat{
					count: 10,
				},
				urls: &stat{
					count: 20,
				},
			},
			expected: &pb.GetStatsRes{Users: 10, Urls: 20},
			wantErr:  false,
		},
		{
			name: "Ошибка запроса количества ссылок",
			urlStore: urlStore{
				urls: &stat{
					err: errors.New("store error"),
				},
			},
			wantErr: true,
			errCode: codes.Internal,
		},
		{
			name: "Ошибка запроса количества пользователей",
			urlStore: urlStore{
				users: &stat{
					err: errors.New("store error"),
				},
				urls: &stat{
					count: 20,
				},
			},
			wantErr: true,
			errCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.urlStore.urls != nil {
				mockStorage.EXPECT().GetURLsCount(gomock.Any()).
					Times(1).Return(tt.urlStore.urls.count, tt.urlStore.urls.err)
			} else {
				mockStorage.EXPECT().GetURLsCount(gomock.Any()).Times(0)
			}
			if tt.urlStore.users != nil {
				mockStorage.EXPECT().GetUsersCount(gomock.Any()).
					Times(1).Return(tt.urlStore.users.count, tt.urlStore.users.err)
			} else {
				mockStorage.EXPECT().GetUsersCount(gomock.Any()).Times(0)
			}

			result, err := server.GetStats(context.Background(), &pb.GetStatsReq{})
			if !tt.wantErr {
				require.NoError(t, err)
				assert.Equal(t, tt.expected.GetUrls(), result.GetUrls())
				assert.Equal(t, tt.expected.GetUsers(), result.GetUsers())
			} else {
				code, _ := status.FromError(err)
				assert.Equal(t, tt.errCode, code.Code())
			}
		})
	}
}

func TestPing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockURLStorage(ctrl)
	baseURL := url.URL{Scheme: "http", Host: "localhost:8080"}
	service := service.NewService(mockStorage, baseURL)
	server := URLShortenerServer{service: &service}

	tests := []struct {
		name     string
		storeErr error
		wantErr  bool
		errCode  codes.Code
	}{
		{
			name:    "Успешный запрос",
			wantErr: false,
		},
		{
			name:     "Ошибка",
			storeErr: errors.New("ping error"),
			wantErr:  true,
			errCode:  codes.Internal,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage.EXPECT().Ping(gomock.Any()).Times(1).Return(tt.storeErr)
			_, err := server.Ping(context.Background(), &pb.PingReq{})
			if !tt.wantErr {
				require.NoError(t, err)
			} else {
				code, _ := status.FromError(err)
				assert.Equal(t, tt.errCode, code.Code())
			}
		})
	}
}
