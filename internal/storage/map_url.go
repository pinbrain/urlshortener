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
	store  sync.Map
	jsonDB jsonDB
}

type jsonDB struct {
	file    *os.File
	encoder *json.Encoder
	decoder *json.Decoder
}

type URLMapFileRecord struct {
	OriginalURL string `json:"original_url"`
	ShortURL    string `json:"short_url"`
}

func NewURLMapStore(storageFile string) (*URLMapStore, error) {
	urlMapStore := &URLMapStore{
		store: sync.Map{},
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
			urlMapStore.store.Store(record.ShortURL, record.OriginalURL)
		}
	}

	return urlMapStore, nil
}

func (s *URLMapStore) SaveURL(_ context.Context, url string) (string, error) {
	id := utils.NewRandomString(urlIDLength)
	if s.jsonDB.file != nil {
		if err := s.jsonDB.encoder.Encode(URLMapFileRecord{OriginalURL: url, ShortURL: id}); err != nil {
			return "", err
		}
	}
	s.store.Store(id, url)
	return id, nil
}

func (s *URLMapStore) GetURL(_ context.Context, id string) (string, error) {
	storeValue, ok := s.store.Load(id)
	if !ok {
		return "", nil
	}
	url, ok := storeValue.(string)
	if !ok {
		return "", errors.New("wrong data type in url store")
	}
	return url, nil
}

func (s *URLMapStore) IsValidID(id string) bool {
	regStr := fmt.Sprintf(`^[a-zA-Z0-9]{%d}$`, urlIDLength)
	validIDReg := regexp.MustCompile(regStr)
	return validIDReg.MatchString(id)
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
