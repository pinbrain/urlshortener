package handlers_test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/pinbrain/urlshortener/internal/handlers"
	"github.com/pinbrain/urlshortener/internal/storage"
)

func ExampleURLHandler_HandleJSONShortenURL() {
	baseURL, err := url.Parse("http://localhost:8080/")
	if err != nil {
		log.Fatal(err)
	}
	mockStorage, err := storage.NewURLStorage(context.Background(), storage.URLStorageConfig{})
	if err != nil {
		log.Fatal(err)
	}
	handler := handlers.NewURLHandler(mockStorage, *baseURL)

	reqBody := `{"url":"http://example.com"}`
	request := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(reqBody))
	request.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.HandleJSONShortenURL(w, request)

	res := w.Result()
	defer res.Body.Close()

	fmt.Println("Status Code:", res.StatusCode)
	// Output:
	// Status Code: 201
}

func ExampleURLHandler_HandleShortenBatchURL() {
	baseURL, err := url.Parse("http://localhost:8080/")
	if err != nil {
		log.Fatal(err)
	}
	mockStorage, err := storage.NewURLStorage(context.Background(), storage.URLStorageConfig{})
	if err != nil {
		log.Fatal(err)
	}
	handler := handlers.NewURLHandler(mockStorage, *baseURL)

	reqBody := `[
								{
									"CorrelationID": "1",
									"OriginalURL": "http://example.com"
								},
								{
									"CorrelationID": "2",
									"OriginalURL": "http://test.com"
								}
							]`

	request := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader(reqBody))
	request.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.HandleShortenBatchURL(w, request)

	res := w.Result()
	defer res.Body.Close()

	fmt.Println("Status Code:", res.StatusCode)
	// Output:
	// Status Code: 201
}

func ExampleURLHandler_HandleRedirect() {
	baseURL, err := url.Parse("http://localhost:8080/")
	if err != nil {
		log.Fatal(err)
	}
	mockStorage, err := storage.NewURLStorage(context.Background(), storage.URLStorageConfig{})
	if err != nil {
		log.Fatal(err)
	}
	handler := handlers.NewURLHandler(mockStorage, *baseURL)

	shortURL, err := mockStorage.SaveURL(context.Background(), "http://example.com", 1)
	if err != nil {
		log.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/"+shortURL, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("urlID", shortURL)
	request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.HandleRedirect(w, request)

	res := w.Result()
	defer res.Body.Close()

	fmt.Println("Status Code:", res.StatusCode)
	fmt.Println("Location Header:", res.Header.Get("Location"))
	// Output:
	// Status Code: 307
	// Location Header: http://example.com
}
