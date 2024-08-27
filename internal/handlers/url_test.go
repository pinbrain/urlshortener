package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pinbrain/urlshortener/internal/middleware"
	"github.com/pinbrain/urlshortener/internal/storage"
	"github.com/pinbrain/urlshortener/internal/storage/mocks"
)

func TestURLHandler_HandleShortenURL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockURLStorage(ctrl)

	type want struct {
		statusCode int
		resBody    string
	}
	type request struct {
		url         string
		contentType string
	}
	type urlStore struct {
		urlID         string
		urlStoreError error
	}
	tests := []struct {
		name     string
		baseURL  string
		want     want
		request  request
		urlStore *urlStore
	}{
		{
			name:    "Успешный запрос",
			baseURL: "http://localhost:8080/",
			request: request{
				url:         "http://some.host.ru",
				contentType: "text/plain",
			},
			urlStore: &urlStore{
				urlID:         "AbCd1234",
				urlStoreError: nil,
			},
			want: want{
				statusCode: http.StatusCreated,
				resBody:    "http://localhost:8080/AbCd1234",
			},
		},
		{
			name:    "Некорректный тип передаваемых данных",
			baseURL: "http://localhost:8080/",
			request: request{
				url:         "http://some.host.ru",
				contentType: "application/json",
			},
			urlStore: nil,
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:    "Некорректная ссылка для сокращения",
			baseURL: "http://localhost:8080/",
			request: request{
				url:         "some random text",
				contentType: "text/plain",
			},
			urlStore: nil,
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:    "Ошибка сохранения записи (ошибка store)",
			baseURL: "http://localhost:8080/",
			request: request{
				url:         "http://some.host.ru",
				contentType: "text/plain",
			},
			urlStore: &urlStore{
				urlStoreError: errors.New("URL store error"),
			},
			want: want{
				statusCode: http.StatusInternalServerError,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseURL, err := url.Parse(tt.baseURL)
			require.NoError(t, err)
			handler := NewURLHandler(mockStorage, *baseURL)

			if tt.urlStore != nil {
				mockStorage.EXPECT().
					SaveURL(gomock.Any(), tt.request.url, gomock.Any()).
					Times(1).
					Return(tt.urlStore.urlID, tt.urlStore.urlStoreError)
			} else {
				mockStorage.EXPECT().SaveURL(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			}

			reqBody := strings.NewReader(tt.request.url)
			request := httptest.NewRequest(http.MethodPost, "/", reqBody)
			request.Header.Set("Content-Type", tt.request.contentType)

			w := httptest.NewRecorder()

			handler.HandleShortenURL(w, request)

			res := w.Result()
			assert.Equal(t, tt.want.statusCode, res.StatusCode)

			if tt.want.resBody != "" {
				defer res.Body.Close()
				resBody, readErr := io.ReadAll(res.Body)
				require.NoError(t, readErr)
				assert.Equal(t, tt.want.resBody, string(resBody))
			}
		})
	}
}

func TestURLHandler_HandleJSONShortenURL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockURLStorage(ctrl)

	type want struct {
		statusCode int
		resBody    string
	}
	type request struct {
		url         string
		contentType string
	}
	type urlStore struct {
		urlID         string
		urlStoreError error
	}
	tests := []struct {
		name     string
		baseURL  string
		want     want
		request  request
		urlStore *urlStore
	}{
		{
			name:    "Успешный запрос",
			baseURL: "http://localhost:8080/",
			request: request{
				url:         "http://some.host.ru",
				contentType: "application/json",
			},
			urlStore: &urlStore{
				urlID:         "AbCd1234",
				urlStoreError: nil,
			},
			want: want{
				statusCode: http.StatusCreated,
				resBody:    `{"result":"http://localhost:8080/AbCd1234"}`,
			},
		},
		{
			name:    "Некорректный тип передаваемых данных",
			baseURL: "http://localhost:8080/",
			request: request{
				url:         "http://some.host.ru",
				contentType: "text/plain",
			},
			urlStore: nil,
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:    "Некорректная ссылка для сокращения",
			baseURL: "http://localhost:8080/",
			request: request{
				url:         "some random text",
				contentType: "application/json",
			},
			urlStore: nil,
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:    "Ошибка сохранения записи (ошибка store)",
			baseURL: "http://localhost:8080/",
			request: request{
				url:         "http://some.host.ru",
				contentType: "application/json",
			},
			urlStore: &urlStore{
				urlStoreError: errors.New("URL store error"),
			},
			want: want{
				statusCode: http.StatusInternalServerError,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseURL, err := url.Parse(tt.baseURL)
			require.NoError(t, err)
			handler := NewURLHandler(mockStorage, *baseURL)

			if tt.urlStore != nil {
				mockStorage.EXPECT().
					SaveURL(gomock.Any(), tt.request.url, gomock.Any()).
					Times(1).
					Return(tt.urlStore.urlID, tt.urlStore.urlStoreError)
			} else {
				mockStorage.EXPECT().SaveURL(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			}

			req := shortenRequest{
				URL: tt.request.url,
			}
			reqJSON, err := json.Marshal(req)
			require.NoError(t, err)
			request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(reqJSON))
			request.Header.Set("Content-Type", tt.request.contentType)

			w := httptest.NewRecorder()

			handler.HandleJSONShortenURL(w, request)

			res := w.Result()
			assert.Equal(t, tt.want.statusCode, res.StatusCode)

			if tt.want.resBody != "" {
				defer res.Body.Close()
				resBody, readErr := io.ReadAll(res.Body)
				require.NoError(t, readErr)
				assert.JSONEq(t, tt.want.resBody, string(resBody))
			}
		})
	}
}

func TestURLHandler_HandleRedirect(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockURLStorage(ctrl)

	type want struct {
		statusCode int
		location   string
	}
	type request struct {
		reqURL string
		urlID  string
	}
	type urlStore struct {
		url           string
		urlStoreError error
	}
	tests := []struct {
		name      string
		request   request
		want      want
		urlStore  *urlStore
		isValidID bool
	}{
		{
			name: "Успешный запрос",
			request: request{
				reqURL: "/AbCd1234",
				urlID:  "AbCd1234",
			},
			want: want{
				statusCode: http.StatusTemporaryRedirect,
				location:   "http://some.host.ru",
			},
			urlStore: &urlStore{
				url:           "http://some.host.ru",
				urlStoreError: nil,
			},
			isValidID: true,
		},
		{
			name: "Некорректный ID ссылки",
			request: request{
				reqURL: "/AbCd1234",
				urlID:  "AbCd1234",
			},
			want: want{
				statusCode: http.StatusBadRequest,
			},
			urlStore:  nil,
			isValidID: false,
		},
		{
			name: "Ошибка чтения записи (ошибка store)",
			request: request{
				reqURL: "/AbCd1234",
				urlID:  "AbCd1234",
			},
			want: want{
				statusCode: http.StatusInternalServerError,
			},
			urlStore: &urlStore{
				urlStoreError: errors.New("URL store error"),
			},
			isValidID: true,
		},
		{
			name: "Сокращенная ссылка не найдена",
			request: request{
				reqURL: "/AbCd1234",
				urlID:  "AbCd1234",
			},
			want: want{
				statusCode: http.StatusNotFound,
			},
			urlStore: &urlStore{
				url:           "",
				urlStoreError: nil,
			},
			isValidID: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewURLHandler(mockStorage, url.URL{
				Scheme: "http",
				Host:   "localhost:8080",
			})

			if tt.urlStore != nil {
				mockStorage.EXPECT().
					GetURL(gomock.Any(), tt.request.urlID).
					Times(1).
					Return(tt.urlStore.url, tt.urlStore.urlStoreError)
			} else {
				mockStorage.EXPECT().GetURL(gomock.Any(), gomock.Any()).Times(0)
			}

			mockStorage.EXPECT().
				IsValidID(tt.request.urlID).
				Times(1).
				Return(tt.isValidID)

			request := httptest.NewRequest(http.MethodGet, tt.request.reqURL, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("urlID", tt.request.urlID)
			request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			handler.HandleRedirect(w, request)

			res := w.Result()
			defer res.Body.Close()
			assert.Equal(t, tt.want.statusCode, res.StatusCode)
			assert.Equal(t, tt.want.location, res.Header.Get("Location"))
		})
	}
}

func TestURLHandler_HandleShortenBatchURL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockURLStorage(ctrl)

	type want struct {
		statusCode int
		resBody    string
	}
	type request struct {
		body        []batchShortenRequest
		contentType string
	}
	type urlStore struct {
		storeError error
		shortURLs  []storage.ShortenURL
	}
	tests := []struct {
		name     string
		baseURL  string
		want     want
		request  request
		urlStore *urlStore
	}{
		{
			name:    "Успешный запрос",
			baseURL: "http://localhost:8080/",
			request: request{
				body: []batchShortenRequest{
					{CorrelationID: "1", OriginalURL: "http://some.host.ru/1"},
					{CorrelationID: "2", OriginalURL: "http://some.host.ru/2"},
				},
				contentType: "application/json",
			},
			urlStore: &urlStore{
				storeError: nil,
				shortURLs: []storage.ShortenURL{
					{Shorten: "AbCd1234"},
					{Shorten: "EfGh5678"},
				},
			},
			want: want{
				statusCode: http.StatusCreated,
				resBody: `
					[
						{
							"correlation_id": "1",
							"short_url": "http://localhost:8080/AbCd1234"
						},
						{
							"correlation_id": "2",
							"short_url": "http://localhost:8080/EfGh5678"
						}
					]
				`,
			},
		},
		{
			name:    "Ошибка при сохранении",
			baseURL: "http://localhost:8080/",
			request: request{
				body: []batchShortenRequest{
					{CorrelationID: "1", OriginalURL: "http://some.host.ru/1"},
				},
				contentType: "application/json",
			},
			urlStore: &urlStore{
				storeError: errors.New("URL store error"),
			},
			want: want{
				statusCode: http.StatusInternalServerError,
			},
		},
		{
			name:    "Повтор ссылок",
			baseURL: "http://localhost:8080/",
			request: request{
				body: []batchShortenRequest{
					{CorrelationID: "1", OriginalURL: "http://some.host.ru/1"},
				},
				contentType: "application/json",
			},
			urlStore: &urlStore{
				storeError: storage.ErrConflict,
			},
			want: want{
				statusCode: http.StatusConflict,
			},
		},
		{
			name:    "Отсутствуют данные для сокращения",
			baseURL: "http://localhost:8080/",
			request: request{
				body:        []batchShortenRequest{},
				contentType: "application/json",
			},
			urlStore: nil,
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseURL, err := url.Parse(tt.baseURL)
			require.NoError(t, err)
			handler := NewURLHandler(mockStorage, *baseURL)

			if tt.urlStore != nil {
				if tt.urlStore.storeError != nil {
					mockStorage.EXPECT().
						SaveBatchURL(gomock.Any(), gomock.Any(), gomock.Any()).
						Times(1).
						Return(tt.urlStore.storeError)
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

			reqBody, err := json.Marshal(tt.request.body)
			require.NoError(t, err)
			request := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(reqBody))
			request.Header.Set("Content-Type", tt.request.contentType)

			w := httptest.NewRecorder()

			handler.HandleShortenBatchURL(w, request)

			res := w.Result()
			assert.Equal(t, tt.want.statusCode, res.StatusCode)

			if tt.want.resBody != "" {
				defer res.Body.Close()
				resBody, readErr := io.ReadAll(res.Body)
				require.NoError(t, readErr)
				assert.JSONEq(t, tt.want.resBody, string(resBody))
			}
		})
	}
}

func TestURLHandler_HandleGetUsersURLs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockURLStorage(ctrl)

	urlHandler := NewURLHandler(mockStorage, url.URL{Scheme: "http", Host: "localhost:8080"})
	router := NewURLRouter(urlHandler, mockStorage)

	user := &storage.User{ID: 1}
	jwtString, err := middleware.BuildJWTString(user.ID)
	require.NoError(t, err)

	type want struct {
		statusCode int
		resBody    string
	}
	type urlStore struct {
		userURLs   []storage.ShortenURL
		storeError error
	}
	tests := []struct {
		name     string
		want     want
		urlStore *urlStore
	}{
		{
			name: "Успешный запрос",
			urlStore: &urlStore{
				userURLs: []storage.ShortenURL{
					{Original: "http://some.host.ru/1", Shorten: "AbCd1234"},
					{Original: "http://some.host.ru/2", Shorten: "EfGh5678"},
				},
				storeError: nil,
			},
			want: want{
				statusCode: http.StatusOK,
				resBody: `
					[
						{
							"original_url": "http://some.host.ru/1",
							"short_url": "http://localhost:8080/AbCd1234"
						},
						{
							"original_url": "http://some.host.ru/2",
							"short_url": "http://localhost:8080/EfGh5678"
						}
					]
				`,
			},
		},
		{
			name: "Ошибка при получении",
			urlStore: &urlStore{
				userURLs:   nil,
				storeError: errors.New("URL store error"),
			},
			want: want{
				statusCode: http.StatusInternalServerError,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage.EXPECT().
				GetUserURLs(gomock.Any(), gomock.Any()).
				Times(1).
				Return(tt.urlStore.userURLs, tt.urlStore.storeError)

			mockStorage.EXPECT().
				GetUser(gomock.Any(), user.ID).
				Times(1).
				Return(user, nil)

			request := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
			request.AddCookie(&http.Cookie{Name: middleware.JWTCookieName, Value: jwtString})
			w := httptest.NewRecorder()

			router.ServeHTTP(w, request)

			res := w.Result()
			assert.Equal(t, tt.want.statusCode, res.StatusCode)

			if tt.want.resBody != "" {
				defer res.Body.Close()
				resBody, readErr := io.ReadAll(res.Body)
				require.NoError(t, readErr)
				assert.JSONEq(t, tt.want.resBody, string(resBody))
			}
		})
	}
}

func TestURLHandler_HandleDeleteUserURLs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockURLStorage(ctrl)

	urlHandler := NewURLHandler(mockStorage, url.URL{Scheme: "http", Host: "localhost:8080"})
	router := NewURLRouter(urlHandler, mockStorage)

	user := &storage.User{ID: 1}
	jwtString, err := middleware.BuildJWTString(user.ID)
	require.NoError(t, err)

	type want struct {
		statusCode int
	}
	tests := []struct {
		name        string
		contentType string
		body        []string
		want        want
		isAuth      bool
	}{
		{
			name:        "Успешный запрос",
			contentType: "application/json",
			body:        []string{"AbCd1234", "EfGh5678"},
			want: want{
				statusCode: http.StatusAccepted,
			},
			isAuth: true,
		},
		{
			name:        "Некорректный тип данных",
			contentType: "text/plain",
			body:        nil,
			want: want{
				statusCode: http.StatusBadRequest,
			},
			isAuth: true,
		},
		{
			name:        "Отсутствуют данные",
			contentType: "application/json",
			body:        []string{},
			want: want{
				statusCode: http.StatusBadRequest,
			},
			isAuth: true,
		},
		{
			name:        "Неавторизованный запрос",
			contentType: "application/json",
			body:        nil,
			want: want{
				statusCode: http.StatusUnauthorized,
			},
			isAuth: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody, mErr := json.Marshal(tt.body)
			require.NoError(t, mErr)
			request := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewReader(reqBody))

			if tt.body != nil && len(tt.body) > 0 {
				mockStorage.EXPECT().
					DeleteUserURLs(gomock.Any(), gomock.Any()).
					Times(1).
					Return(nil)
			}

			if tt.isAuth {
				mockStorage.EXPECT().
					GetUser(gomock.Any(), user.ID).
					Times(1).
					Return(user, nil)
				request.AddCookie(&http.Cookie{Name: middleware.JWTCookieName, Value: jwtString})
			} else {
				mockStorage.EXPECT().
					CreateUser(gomock.Any()).
					Times(1).
					Return(&storage.User{ID: 2}, nil)
			}
			request.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, request)

			res := w.Result()
			time.Sleep(500 * time.Millisecond)
			defer res.Body.Close()
			assert.Equal(t, tt.want.statusCode, res.StatusCode)
		})
	}
}

func TestURLHandler_HandlePing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockURLStorage(ctrl)

	type want struct {
		statusCode int
	}
	tests := []struct {
		name      string
		want      want
		pingError error
	}{
		{
			name: "Успешный пинг",
			want: want{
				statusCode: http.StatusOK,
			},
			pingError: nil,
		},
		{
			name: "Ошибка пинга",
			want: want{
				statusCode: http.StatusInternalServerError,
			},
			pingError: errors.New("Ping failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewURLHandler(mockStorage, url.URL{})

			mockStorage.EXPECT().
				Ping(gomock.Any()).
				Times(1).
				Return(tt.pingError)

			request := httptest.NewRequest(http.MethodGet, "/ping", nil)
			w := httptest.NewRecorder()

			handler.HandlePing(w, request)

			res := w.Result()
			defer res.Body.Close()
			assert.Equal(t, tt.want.statusCode, res.StatusCode)
		})
	}
}
