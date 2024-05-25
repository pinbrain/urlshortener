package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pinbrain/urlshortener/internal/logger"
	"github.com/pinbrain/urlshortener/internal/utils"
)

type URLStorage interface {
	SaveURL(url string) (id string, err error)
	GetURL(id string) (url string, err error)
	IsValidID(id string) bool
}

type URLHandler struct {
	urlStore URLStorage
	baseURL  *url.URL
}

type shortenRequest struct {
	URL string `json:"url"`
}

type shortenResponse struct {
	Result string `json:"result"`
}

func NewURLHandler(urlStore URLStorage, baseURL url.URL) URLHandler {
	return URLHandler{
		urlStore: urlStore,
		baseURL:  &baseURL,
	}
}

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
	urlID, err := h.urlStore.SaveURL(url)
	if err != nil {
		logger.Log.Errorw("Error while saving url for shorten", "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	shortURL := h.baseURL.JoinPath(urlID).String()
	w.WriteHeader(http.StatusCreated)
	if _, err = w.Write([]byte(shortURL)); err != nil {
		logger.Log.Errorw("Error sending response", "err", err)
	}
}

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
	urlID, err := h.urlStore.SaveURL(req.URL)
	if err != nil {
		logger.Log.Errorw("Error while saving url for shorten", "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	shortURL := h.baseURL.JoinPath(urlID).String()

	resp := shortenResponse{
		Result: shortURL,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	enc := json.NewEncoder(w)
	if err = enc.Encode(resp); err != nil {
		logger.Log.Errorw("Error in encoding shorten response to json", "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (h *URLHandler) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	urlID := chi.URLParam(r, "urlID")
	if !h.urlStore.IsValidID(urlID) {
		http.Error(w, "Некорректная ссылка", http.StatusBadRequest)
		return
	}
	url, err := h.urlStore.GetURL(urlID)
	if err != nil {
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
