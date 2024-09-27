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

	"github.com/pinbrain/urlshortener/internal/context"
	"github.com/pinbrain/urlshortener/internal/logger"
	"github.com/pinbrain/urlshortener/internal/storage"
	"github.com/pinbrain/urlshortener/internal/utils"
)

// URLHandler определяет структуру обработчика запросов сервиса.
type URLHandler struct {
	urlStore storage.URLStorage // Хранилище приложения
	baseURL  *url.URL           // Базовый url сокращаемых ссылок
	wg       *sync.WaitGroup    // Waiting group для go рутин хендлера
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
func NewURLHandler(urlStore storage.URLStorage, baseURL url.URL) URLHandler {
	return URLHandler{
		urlStore: urlStore,
		baseURL:  &baseURL,
		wg:       &sync.WaitGroup{},
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
	isValidURL := utils.IsValidURLString(url)
	if !isValidURL {
		http.Error(w, "Некорректная ссылка для сокращения", http.StatusBadRequest)
		return
	}
	user := context.GetCtxUser(r.Context())
	userID := 0
	if user != nil {
		userID = user.ID
	}
	urlID, err := h.urlStore.SaveURL(r.Context(), url, userID)
	if err != nil {
		if !errors.Is(err, storage.ErrConflict) {
			logger.Log.Errorw("Error while saving url for shorten", "err", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusConflict)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	shortURL := h.baseURL.JoinPath(urlID).String()
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
	isValidURL := utils.IsValidURLString(req.URL)
	if !isValidURL {
		http.Error(w, "Некорректная ссылка для сокращения", http.StatusBadRequest)
		return
	}
	user := context.GetCtxUser(r.Context())
	userID := 0
	if user != nil {
		userID = user.ID
	}
	urlID, err := h.urlStore.SaveURL(r.Context(), req.URL, userID)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		if !errors.Is(err, storage.ErrConflict) {
			logger.Log.Errorw("Error while saving url for shorten", "err", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusConflict)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	shortURL := h.baseURL.JoinPath(urlID).String()

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
	if len(req) == 0 {
		http.Error(w, "Отсутствуют данные для сокращения", http.StatusBadRequest)
		return
	}

	shortenURLs := []storage.ShortenURL{}
	for _, reqURL := range req {
		shortenURLs = append(shortenURLs, storage.ShortenURL{
			Original: reqURL.OriginalURL,
		})
	}
	user := context.GetCtxUser(r.Context())
	userID := 0
	if user != nil {
		userID = user.ID
	}
	err := h.urlStore.SaveBatchURL(r.Context(), shortenURLs, userID)
	if err != nil {
		if errors.Is(err, storage.ErrConflict) {
			http.Error(w, "В запросе ссылки, которые уже были ранее сохранены", http.StatusConflict)
			return
		}
		logger.Log.Errorw("Error in saving batch of urls in store", "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	resp := []batchShortenResponse{}
	for i, url := range req {
		result := batchShortenResponse{
			CorrelationID: url.CorrelationID,
			ShortURL:      h.baseURL.JoinPath(shortenURLs[i].Shorten).String(),
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
	user := context.GetCtxUser(r.Context())
	userURLs, err := h.urlStore.GetUserURLs(r.Context(), user.ID)
	if err != nil {
		logger.Log.Errorw("Error in getting user shorten urls", "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	resp := []userURLResponse{}
	for _, url := range userURLs {
		result := userURLResponse{
			OriginalURL: url.Original,
			ShortURL:    h.baseURL.JoinPath(url.Shorten).String(),
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
	user := context.GetCtxUser(r.Context())
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
		h.urlStore.DeleteUserURLs(user.ID, req)
	}()
	w.WriteHeader(http.StatusAccepted)
}

// HandleRedirect обрабатывает запрос переход по сокращенной ссылке.
func (h *URLHandler) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	urlID := chi.URLParam(r, "urlID")
	if !h.urlStore.IsValidID(urlID) {
		http.Error(w, "Некорректная ссылка", http.StatusBadRequest)
		return
	}
	url, err := h.urlStore.GetURL(r.Context(), urlID)
	if err != nil {
		if errors.Is(err, storage.ErrIsDeleted) {
			w.WriteHeader(http.StatusGone)
			return
		}
		logger.Log.Errorw("Error getting shorten url", "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if url == "" {
		http.Error(w, "Сокращенная ссылка не найдена", http.StatusNotFound)
		return
	}
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// HandlePing обрабатывает запрос на проверку соединения с хранилищем данных.
func (h *URLHandler) HandlePing(w http.ResponseWriter, r *http.Request) {
	if err := h.urlStore.Ping(r.Context()); err != nil {
		logger.Log.Errorw("Error trying to ping db", "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// HandleGetStats обрабатывает запрос на получение статистики хранилища.
func (h *URLHandler) HandleGetStats(w http.ResponseWriter, r *http.Request) {
	var err error
	stats := statsResponse{}
	stats.URLs, err = h.urlStore.GetURLsCount(r.Context())
	if err != nil {
		logger.Log.Errorw("Error trying to get urls count", "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
	stats.Users, err = h.urlStore.GetUsersCount(r.Context())
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
