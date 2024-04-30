package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pinbrain/urlshortener/internal/handlers/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestURLHandler_HandleShortenURL(t *testing.T) {
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
		urlStore urlStore
	}{
		{
			name:    "Успешный запрос",
			baseURL: "http://localhost:8080/",
			request: request{
				url:         "http://some.host.ru",
				contentType: "text/plain",
			},
			urlStore: urlStore{
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
			urlStore: urlStore{
				urlStoreError: nil,
			},
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
			urlStore: urlStore{
				urlStoreError: nil,
			},
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
			urlStore: urlStore{
				urlStoreError: errors.New("URL store error"),
			},
			want: want{
				statusCode: http.StatusInternalServerError,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := new(mocks.MockURLStorage)
			baseURL, err := url.Parse(tt.baseURL)
			require.NoError(t, err)
			handler := NewURLHandler(mockStorage, *baseURL)

			mockStorage.On("SaveURL", tt.request.url).Return(tt.urlStore.urlID, tt.urlStore.urlStoreError)

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
func TestURLHandler_HandleRedirect(t *testing.T) {
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
		isValidID     bool
	}
	tests := []struct {
		name     string
		request  request
		want     want
		urlStore urlStore
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
			urlStore: urlStore{
				url:           "http://some.host.ru",
				urlStoreError: nil,
				isValidID:     true,
			},
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
			urlStore: urlStore{
				isValidID: false,
			},
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
			urlStore: urlStore{
				urlStoreError: errors.New("URL store error"),
				isValidID:     true,
			},
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
			urlStore: urlStore{
				url:           "",
				urlStoreError: nil,
				isValidID:     true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := new(mocks.MockURLStorage)
			// handler, err := NewURLHandler(mockStorage, "http://localhost:8080/")
			handler := NewURLHandler(mockStorage, url.URL{
				Scheme: "http",
				Host:   "localhost:8080",
			})

			mockStorage.On("GetURL", tt.request.urlID).Return(tt.urlStore.url, tt.urlStore.urlStoreError)
			mockStorage.On("IsValidID", tt.request.urlID).Return(tt.urlStore.isValidID)

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
