package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sync"

	"github.com/pinbrain/urlshortener/internal/utils"
)

type URLMapStore struct {
	store     map[string]string
	userStore map[int][]string
	userMaxID int
	mutex     sync.RWMutex
	jsonDB    jsonDB
}

type jsonDB struct {
	file    *os.File
	encoder *json.Encoder
	decoder *json.Decoder
}

type URLMapFileRecord struct {
	OriginalURL string `json:"original_url"`
	ShortURL    string `json:"short_url"`
	UserID      int    `json:"user_id"`
}

func NewURLMapStore(storageFile string) (*URLMapStore, error) {
	urlMapStore := &URLMapStore{
		store:     make(map[string]string),
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
			urlMapStore.store[record.ShortURL] = record.OriginalURL
			userURLs := urlMapStore.userStore[record.UserID]
			urlMapStore.userStore[record.UserID] = append(userURLs, record.ShortURL)
			if urlMapStore.userMaxID < record.UserID {
				urlMapStore.userMaxID = record.UserID
			}
		}
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
	s.store[id] = url
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
	url, ok := s.store[id]
	if !ok {
		return "", nil
	}
	return url, nil
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
		userURLs = append(userURLs, ShortenURL{
			Shorten:  url,
			Original: s.store[url],
		})
	}
	return userURLs, nil
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
