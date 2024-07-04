package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/pinbrain/urlshortener/internal/logger"
	"github.com/pinbrain/urlshortener/internal/utils"
)

type URLMapStore struct {
	store     map[string]URLMapData
	userStore map[int][]string
	userMaxID int
	mutex     sync.RWMutex
	jsonDB    jsonDB
}

type jsonDB struct {
	file         *os.File
	encoder      *json.Encoder
	decoder      *json.Decoder
	needSyncFile bool
}

type URLMapFileRecord struct {
	OriginalURL string `json:"original_url"`
	ShortURL    string `json:"short_url"`
	UserID      int    `json:"user_id"`
	IsDeleted   bool   `json:"is_deleted"`
}

type URLMapData struct {
	OriginalURL string
	UserID      int
	IsDeleted   bool
}

const syncFileInterval = 30

func NewURLMapStore(ctx context.Context, storageFile string) (*URLMapStore, error) {
	urlMapStore := &URLMapStore{
		store:     make(map[string]URLMapData),
		userStore: make(map[int][]string),
	}

	if storageFile != "" {
		jsonDBFile, err := os.OpenFile(storageFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}
		urlMapStore.jsonDB = jsonDB{
			file:    jsonDBFile,
			encoder: json.NewEncoder(jsonDBFile),
			decoder: json.NewDecoder(jsonDBFile),
		}

		record := &URLMapFileRecord{}
		for {
			if err = urlMapStore.jsonDB.decoder.Decode(record); err != nil {
				if err.Error() == "EOF" {
					break
				}
				return nil, err
			}
			urlMapStore.store[record.ShortURL] = URLMapData{
				OriginalURL: record.OriginalURL,
				IsDeleted:   record.IsDeleted,
				UserID:      record.UserID,
			}
			userURLs := urlMapStore.userStore[record.UserID]
			urlMapStore.userStore[record.UserID] = append(userURLs, record.ShortURL)
			if urlMapStore.userMaxID < record.UserID {
				urlMapStore.userMaxID = record.UserID
			}
		}

		go urlMapStore.syncFileData(ctx)
	}

	return urlMapStore, nil
}

func (s *URLMapStore) SaveURL(_ context.Context, url string, userID int) (string, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if userID > s.userMaxID {
		s.userMaxID = userID
	}
	id := utils.NewRandomString(urlIDLength)
	if s.jsonDB.file != nil {
		if err := s.jsonDB.encoder.Encode(URLMapFileRecord{OriginalURL: url, ShortURL: id, UserID: userID}); err != nil {
			return "", err
		}
	}
	s.store[id] = URLMapData{OriginalURL: url, IsDeleted: false, UserID: userID}
	s.userStore[userID] = append(s.userStore[userID], id)
	return id, nil
}

func (s *URLMapStore) SaveBatchURL(ctx context.Context, urls []ShortenURL, userID int) error {
	for i, url := range urls {
		urlID, err := s.SaveURL(ctx, url.Original, userID)
		if err != nil {
			return fmt.Errorf("failed to save batch of urls: %w", err)
		}
		urls[i].Shorten = urlID
	}
	return nil
}

func (s *URLMapStore) GetURL(_ context.Context, id string) (string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	urlData, ok := s.store[id]
	if !ok {
		return "", nil
	}
	if urlData.IsDeleted {
		return "", ErrIsDeleted
	}
	return urlData.OriginalURL, nil
}

func (s *URLMapStore) IsValidID(id string) bool {
	regStr := fmt.Sprintf(`^[a-zA-Z0-9]{%d}$`, urlIDLength)
	validIDReg := regexp.MustCompile(regStr)
	return validIDReg.MatchString(id)
}

func (s *URLMapStore) CreateUser(_ context.Context) (*User, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.userMaxID++
	return &User{ID: s.userMaxID}, nil
}

func (s *URLMapStore) GetUser(_ context.Context, id int) (*User, error) {
	if id <= 0 {
		return nil, errors.New("invalid user id")
	}
	_, ok := s.userStore[id]
	if !ok {
		return nil, ErrNoData
	}
	return &User{ID: id}, nil
}

func (s *URLMapStore) GetUserURLs(_ context.Context, userID int) ([]ShortenURL, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user id")
	}
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var userURLs []ShortenURL
	userStore := s.userStore[userID]
	for _, url := range userStore {
		if s.store[url].IsDeleted {
			continue
		}
		userURLs = append(userURLs, ShortenURL{
			Shorten:  url,
			Original: s.store[url].OriginalURL,
		})
	}
	return userURLs, nil
}

func (s *URLMapStore) DeleteUserURLs(userID int, urls []string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, url := range urls {
		urlData, ok := s.store[url]
		if !ok || urlData.IsDeleted || urlData.UserID != userID {
			continue
		}
		s.store[url] = URLMapData{OriginalURL: s.store[url].OriginalURL, IsDeleted: true}
		s.jsonDB.needSyncFile = true
	}
	return nil
}

func (s *URLMapStore) processSyncFileData() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.jsonDB.file != nil {
		logger.Log.Debugln("starting file sync")
		// Временный файл для копирования в него всех данных
		tmpFile, err := os.CreateTemp(".", "jsondb-*.tmp")
		defer func() {
			if err = tmpFile.Close(); err != nil {
			}
		}()
		if err != nil {
			return fmt.Errorf("failed to create temporary file: %w", err)
		}

		tmpEncoder := json.NewEncoder(tmpFile)

		// Все записи из памяти переносим во временный файл
		for shortURL, data := range s.store {
			record := URLMapFileRecord{
				OriginalURL: data.OriginalURL,
				ShortURL:    shortURL,
				UserID:      data.UserID,
				IsDeleted:   data.IsDeleted,
			}
			if err = tmpEncoder.Encode(&record); err != nil {
				return fmt.Errorf("failed to encode record to temporary file: %w", err)
			}
		}

		// Закрываем файлы для замены старого на новый
		if err = tmpFile.Close(); err != nil {
			return fmt.Errorf("failed to close temporary file: %w", err)
		}
		if err = s.jsonDB.file.Close(); err != nil {
			return fmt.Errorf("failed to close the original file: %w", err)
		}
		if err = os.Rename(tmpFile.Name(), s.jsonDB.file.Name()); err != nil {
			return fmt.Errorf("failed to replace the original file: %w", err)
		}

		// Переоткрываем новый файл для использования хранилищем
		s.jsonDB.file, err = os.OpenFile(s.jsonDB.file.Name(), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			return fmt.Errorf("failed to reopen the original file: %w", err)
		}
		s.jsonDB.encoder = json.NewEncoder(s.jsonDB.file)
		s.jsonDB.decoder = json.NewDecoder(s.jsonDB.file)
		s.jsonDB.needSyncFile = false
	}

	return nil
}

func (s *URLMapStore) syncFileData(ctx context.Context) {
	ticker := time.NewTicker(syncFileInterval * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if s.jsonDB.needSyncFile {
				if err := s.processSyncFileData(); err != nil {
					logger.Log.Errorw("failed to sync file data", "err", err)
				}
			}
		case <-ctx.Done():
			if err := s.processSyncFileData(); err != nil {
				logger.Log.Errorw("failed to sync file data", "err", err)
			}
			return
		}
	}
}

func (s *URLMapStore) Ping(_ context.Context) error {
	return nil
}

func (s *URLMapStore) Close() error {
	if s.jsonDB.file != nil {
		return s.jsonDB.file.Close()
	}
	return nil
}
