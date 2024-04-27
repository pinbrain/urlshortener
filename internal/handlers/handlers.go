package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pinbrain/urlshortener/internal/storage"
	"github.com/pinbrain/urlshortener/internal/utils"
)

type UrlHandler struct {
	urlStore storage.UrlStorage
	baseUrl  string
}

func NewUrlHandler(urlStore storage.UrlStorage, baseUrl string) UrlHandler {
	return UrlHandler{
		urlStore: urlStore,
		baseUrl:  baseUrl,
	}
}

func (h *UrlHandler) HandleRequest(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.HandleRedirect(w, r)
	case http.MethodPost:
		h.HandleShortenUrl(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}

func (h *UrlHandler) HandleShortenUrl(w http.ResponseWriter, r *http.Request) {
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
	isValidUrl := utils.IsValidUrlString(url)
	if !isValidUrl {
		http.Error(w, "Некорректная ссылка для сокращения", http.StatusBadRequest)
		return
	}
	urlID, err := h.urlStore.SaveUrl(url)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	shortUrl := h.baseUrl + urlID
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortUrl))
}

func (h *UrlHandler) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	urlID := r.URL.Path[1:]
	if !h.urlStore.IsValidUrlID(urlID) {
		http.Error(w, "Некорректная ссылка", http.StatusBadRequest)
		return
	}
	url, err := h.urlStore.GetUrl(urlID)
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
