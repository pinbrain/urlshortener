// Package service содержит модуль реализующий бизнес логику
package service

import (
	"context"
	"errors"
	"net/url"

	appCtx "github.com/pinbrain/urlshortener/internal/context"
	"github.com/pinbrain/urlshortener/internal/logger"
	"github.com/pinbrain/urlshortener/internal/storage"
	"github.com/pinbrain/urlshortener/internal/utils"
)

// Ошибки сервиса.
var (
	ErrStorageError  = errors.New("storage error")
	ErrInvalidURL    = errors.New("invalid url string")
	ErrURLConflict   = errors.New("url already exists")
	ErrNoData        = errors.New("no data")
	ErrIsDeleted     = errors.New("data is deleted")
	ErrNotFound      = errors.New("data not found")
	ErrInvalidUserID = errors.New("invalid user id")
)

// URLData описывает структуру данных ссылки (сокращенная и полная).
type URLData struct {
	OriginalURL string
	ShortURL    string
}

// BatchURL описывает структуру данных ссылок при batch запросах.
type BatchURL struct {
	OriginalURL   string
	ShortURL      string
	CorrelationID string
}

// Service описывает структуру сервиса с бизнес логикой.
type Service struct {
	urlStore storage.URLStorage // Хранилище приложения
	baseURL  *url.URL           // Базовый url сокращаемых ссылок
}

// NewService создает и возвращает новый сервис.
func NewService(urlStore storage.URLStorage, baseURL url.URL) Service {
	return Service{
		urlStore: urlStore,
		baseURL:  &baseURL,
	}
}

// ShortenURL сокращает и сохраняет ссылку.
func (s *Service) ShortenURL(ctx context.Context, url string) (string, error) {
	isValidURL := utils.IsValidURLString(url)
	if !isValidURL {
		return "", ErrInvalidURL
	}
	user := appCtx.GetCtxUser(ctx)
	userID := 0
	if user != nil {
		userID = user.ID
	}
	urlID, err := s.urlStore.SaveURL(ctx, url, userID)
	if err != nil {
		if !errors.Is(err, storage.ErrConflict) {
			logger.Log.Errorw("Error while saving url for shorten", "err", err)
			return "", errors.Join(ErrStorageError, err)
		}
		return s.baseURL.JoinPath(urlID).String(), ErrURLConflict
	}
	shortURL := s.baseURL.JoinPath(urlID).String()
	return shortURL, nil
}

// ShortenBatchURL сокращает и сохраняет массив ссылок.
func (s *Service) ShortenBatchURL(ctx context.Context, urls []BatchURL) ([]BatchURL, error) {
	if len(urls) == 0 {
		return nil, ErrNoData
	}
	user := appCtx.GetCtxUser(ctx)
	userID := 0
	if user != nil {
		userID = user.ID
	}

	shortenURLs := []storage.ShortenURL{}
	for _, url := range urls {
		shortenURLs = append(shortenURLs, storage.ShortenURL{Original: url.OriginalURL})
	}
	err := s.urlStore.SaveBatchURL(ctx, shortenURLs, userID)
	if err != nil {
		if !errors.Is(err, storage.ErrConflict) {
			logger.Log.Errorw("Error while saving batch url for shorten", "err", err)
			return nil, errors.Join(ErrStorageError, err)
		}
		return nil, ErrURLConflict
	}
	for i := range urls {
		urls[i].ShortURL = s.baseURL.JoinPath(shortenURLs[i].Shorten).String()
	}
	return urls, nil
}

// GetURL возвращает полную ссылку по id сокращенной.
func (s *Service) GetURL(ctx context.Context, urlID string) (string, error) {
	if !s.urlStore.IsValidID(urlID) {
		return "", ErrInvalidURL
	}
	url, err := s.urlStore.GetURL(ctx, urlID)
	if err != nil {
		if errors.Is(err, storage.ErrIsDeleted) {
			return "", ErrIsDeleted
		}
		logger.Log.Errorw("Error getting shorten url", "err", err)
		return "", errors.Join(ErrStorageError, err)
	}
	if url == "" {
		return "", ErrNotFound
	}
	return url, nil
}

// GetUserURLs возвращает сокращенные ссылки пользователя.
func (s *Service) GetUserURLs(ctx context.Context) ([]URLData, error) {
	user := appCtx.GetCtxUser(ctx)
	if user == nil {
		return nil, nil
	}
	userURLs, err := s.urlStore.GetUserURLs(ctx, user.ID)
	if err != nil {
		logger.Log.Errorw("Error in getting user shorten urls", "err", err)
		return nil, errors.Join(ErrStorageError, err)
	}
	var result []URLData
	for _, url := range userURLs {
		result = append(result, URLData{OriginalURL: url.Original, ShortURL: url.Shorten})
	}
	return result, nil
}

// DeleteUserURLs удаляет сокращенные ссылки пользователя.
func (s *Service) DeleteUserURLs(ctx context.Context, urls []string) error {
	user := appCtx.GetCtxUser(ctx)
	if user == nil {
		return nil
	}
	if len(urls) == 0 {
		return ErrNoData
	}
	err := s.urlStore.DeleteUserURLs(user.ID, urls)
	if err != nil {
		logger.Log.Errorw("Error in deleting user urls", "err", err)
		return errors.Join(ErrStorageError, err)
	}
	return nil
}

// GetUser возвращает данные пользователя по ID.
func (s *Service) GetUser(ctx context.Context, userID int) (*storage.User, error) {
	if userID <= 0 {
		return nil, ErrInvalidUserID
	}
	userData, err := s.urlStore.GetUser(ctx, userID)
	if err != nil {
		if errors.Is(err, storage.ErrNoData) {
			return nil, ErrNotFound
		}
		logger.Log.Errorw("Error getting user data", "err", err)
		return nil, errors.Join(ErrStorageError, err)
	}
	return userData, nil
}

// CreateUser создает нового пользователя.
func (s *Service) CreateUser(ctx context.Context) (*storage.User, error) {
	userData, err := s.urlStore.CreateUser(ctx)
	if err != nil {
		logger.Log.Errorw("Error creating new user", "err", err)
		return nil, errors.Join(ErrStorageError, err)
	}
	return userData, nil
}

// Ping проверяет связь с хранилищем.
func (s *Service) Ping(ctx context.Context) error {
	return s.urlStore.Ping(ctx)
}

// GetURLsCount возвращает общее количество сокращенных ссылок в хранилище.
func (s *Service) GetURLsCount(ctx context.Context) (int, error) {
	return s.urlStore.GetURLsCount(ctx)
}

// GetUsersCount возвращает количество пользователей в хранилище.
func (s *Service) GetUsersCount(ctx context.Context) (int, error) {
	return s.urlStore.GetUsersCount(ctx)
}
