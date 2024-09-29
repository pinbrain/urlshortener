package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/pinbrain/urlshortener/internal/logger"
	"github.com/pinbrain/urlshortener/internal/service"
)

// URLHandler определяет структуру обработчика запросов сервиса.
type URLHandler struct {
	service *service.Service // Сервис с бизнес логикой приложения
	baseURL *url.URL         // Базовый url сокращаемых ссылок
	wg      *sync.WaitGroup  // Waiting group для go рутин хендлера
}

// shortenRequest определяет формат запроса на сокращение ссылки.
type shortenRequest struct {
	URL string `json:"url"`
}

// shortenRequest определяет формат ответа на сокращение ссылки.
type shortenResponse struct {
	Result string `json:"result"`
}

// shortenRequest определяет формат запроса на сокращение нескольких ссылок.
type batchShortenRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

// shortenRequest определяет формат ответа на сокращение нескольких ссылок.
type batchShortenResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

// userURLResponse определяет формат ответа на запрос ссылок, сокращенных пользователем.
type userURLResponse struct {
	OriginalURL string `json:"original_url"` // Исходная ссылка
	ShortURL    string `json:"short_url"`    // Сокращенная ссылка
}

type statsResponse struct {
	URLs  int `json:"urls"`  // количество сокращённых URL в сервисе
	Users int `json:"users"` // количество пользователей в сервисе
}

// NewURLHandler создает и возвращает новый обработчик запросов.
func NewURLHandler(service *service.Service, baseURL url.URL) URLHandler {
	return URLHandler{
		service: service,
		baseURL: &baseURL,
		wg:      &sync.WaitGroup{},
	}
}

// HandleShortenURL обрабатывает запрос на сокращение ссылки (формат тела запроса - строка).
func (h *URLHandler) HandleShortenURL(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") && !strings.Contains(contentType, "application/x-gzip") {
		http.Error(w, "Invalid content type", http.StatusBadRequest)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Log.Errorw("Error in reading request body", "err", err)
		http.Error(w, "Не удалось прочитать ссылку в запросе", http.StatusInternalServerError)
		return
	}
	url := string(body)
	shortURL, err := h.service.ShortenURL(r.Context(), url)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrURLConflict):
			w.WriteHeader(http.StatusConflict)
		case errors.Is(err, service.ErrInvalidURL):
			http.Error(w, "Некорректная ссылка для сокращения", http.StatusBadRequest)
			return
		default:
			logger.Log.Errorw("Error while saving url for shorten", "err", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	if _, err = w.Write([]byte(shortURL)); err != nil {
		logger.Log.Errorw("Error sending response", "err", err)
	}
}

// HandleJSONShortenURL обрабатывает запрос на сокращение ссылки (формат тела запроса - JSON).
func (h *URLHandler) HandleJSONShortenURL(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		http.Error(w, "Invalid content type", http.StatusBadRequest)
		return
	}

	var req shortenRequest
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		logger.Log.Errorw("Error in decoding shorten request body", "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if req.URL == "" {
		http.Error(w, "Отсутствует ссылка для сокращения", http.StatusBadRequest)
		return
	}
	shortURL, err := h.service.ShortenURL(r.Context(), req.URL)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		switch {
		case errors.Is(err, service.ErrURLConflict):
			w.WriteHeader(http.StatusConflict)
		case errors.Is(err, service.ErrInvalidURL):
			http.Error(w, "Некорректная ссылка для сокращения", http.StatusBadRequest)
			return
		default:
			logger.Log.Errorw("Error while saving url for shorten", "err", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	} else {
		w.WriteHeader(http.StatusCreated)
	}

	resp := shortenResponse{
		Result: shortURL,
	}
	enc := json.NewEncoder(w)
	if err = enc.Encode(resp); err != nil {
		logger.Log.Errorw("Error in encoding shorten response to json", "err", err)
	}
}

// HandleShortenBatchURL обрабатывает запрос на сокращение нескольких ссылок.
func (h *URLHandler) HandleShortenBatchURL(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		http.Error(w, "Invalid content type", http.StatusBadRequest)
		return
	}

	var req []batchShortenRequest
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		logger.Log.Errorw("Error in decoding shorten request body", "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	shortenURLs := []service.BatchURL{}
	for _, reqURL := range req {
		shortenURLs = append(shortenURLs, service.BatchURL{
			OriginalURL:   reqURL.OriginalURL,
			CorrelationID: reqURL.CorrelationID,
		})
	}
	savedBatch, err := h.service.ShortenBatchURL(r.Context(), shortenURLs)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrURLConflict):
			http.Error(w, "В запросе ссылки, которые уже были ранее сохранены", http.StatusConflict)
			return
		case errors.Is(err, service.ErrNoData):
			http.Error(w, "Отсутствуют данные для сокращения", http.StatusBadRequest)
			return
		default:
			logger.Log.Errorw("Error in saving batch of urls in store", "err", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	resp := []batchShortenResponse{}
	for _, url := range savedBatch {
		result := batchShortenResponse{
			CorrelationID: url.CorrelationID,
			ShortURL:      url.ShortURL,
		}
		resp = append(resp, result)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	enc := json.NewEncoder(w)
	if err = enc.Encode(resp); err != nil {
		logger.Log.Errorw("Error in encoding shorten response to json", "err", err)
	}
}

// HandleGetUsersURLs обрабатывает запрос на получение ссылок, сокращенных пользователем.
func (h *URLHandler) HandleGetUsersURLs(w http.ResponseWriter, r *http.Request) {
	userURLs, err := h.service.GetUserURLs(r.Context())
	if err != nil {
		logger.Log.Errorw("Error in getting user shorten urls", "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	resp := []userURLResponse{}
	for _, url := range userURLs {
		result := userURLResponse{
			OriginalURL: url.OriginalURL,
			ShortURL:    h.baseURL.JoinPath(url.ShortURL).String(),
		}
		resp = append(resp, result)
	}

	w.Header().Set("Content-Type", "application/json")
	if len(resp) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	enc := json.NewEncoder(w)
	if err = enc.Encode(resp); err != nil {
		logger.Log.Errorw("Error in encoding user urls response to json", "err", err)
	}
}

// HandleDeleteUserURLs обрабатывает запрос на удаление сокращенных ссылок.
func (h *URLHandler) HandleDeleteUserURLs(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		http.Error(w, "Invalid content type", http.StatusBadRequest)
		return
	}

	var req []string
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		logger.Log.Errorw("Error in decoding request body", "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if len(req) == 0 {
		http.Error(w, "Отсутствуют данные для удаления", http.StatusBadRequest)
		return
	}

	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		h.service.DeleteUserURLs(r.Context(), req)
	}()
	w.WriteHeader(http.StatusAccepted)
}

// HandleRedirect обрабатывает запрос переход по сокращенной ссылке.
func (h *URLHandler) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	urlID := chi.URLParam(r, "urlID")
	url, err := h.service.GetURL(r.Context(), urlID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidURL):
			http.Error(w, "Некорректная ссылка", http.StatusBadRequest)
			return
		case errors.Is(err, service.ErrIsDeleted):
			w.WriteHeader(http.StatusGone)
			return
		case errors.Is(err, service.ErrNotFound):
			http.Error(w, "Сокращенная ссылка не найдена", http.StatusNotFound)
			return
		default:
			logger.Log.Errorw("Error getting shorten url", "err", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// HandlePing обрабатывает запрос на проверку соединения с хранилищем данных.
func (h *URLHandler) HandlePing(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Ping(r.Context()); err != nil {
		logger.Log.Errorw("Error trying to ping db", "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// HandleGetStats обрабатывает запрос на получение статистики хранилища.
func (h *URLHandler) HandleGetStats(w http.ResponseWriter, r *http.Request) {
	var err error
	stats := statsResponse{}
	stats.URLs, err = h.service.GetURLsCount(r.Context())
	if err != nil {
		logger.Log.Errorw("Error trying to get urls count", "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
	stats.Users, err = h.service.GetUsersCount(r.Context())
	if err != nil {
		logger.Log.Errorw("Error trying to get users count", "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	if err = enc.Encode(stats); err != nil {
		logger.Log.Errorw("Error in encoding stats response to json", "err", err)
	}
}

// Close дожидается завершения всех goroutines хэндлера.
func (h *URLHandler) Close() {
	logger.Log.Debug("Waiting handler goroutines to finish")
	h.wg.Wait()
}
