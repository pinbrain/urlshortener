package handlers

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
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

func NewURLHandler(urlStore URLStorage, baseURL url.URL) URLHandler {
	return URLHandler{
		urlStore: urlStore,
		baseURL:  &baseURL,
	}
}

func (h *URLHandler) HandleShortenURL(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") {
		http.Error(w, "Invalid content type", http.StatusBadRequest)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println(err)
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
		fmt.Println(err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	shortURL := h.baseURL.JoinPath(urlID).String()
	w.WriteHeader(http.StatusCreated)
	if _, err = w.Write([]byte(shortURL)); err != nil {
		fmt.Println(err)
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
		fmt.Println(err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if url == "" {
		http.Error(w, "Сокращенная ссылка не найдена", http.StatusNotFound)
		return
	}
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}
