package storage

import (
	"errors"
	"fmt"
	"regexp"
	"sync"

	"github.com/pinbrain/urlshortener/internal/utils"
)

const urlIDLength = 8

type URLMapStore struct {
	store sync.Map
}

func NewURLMapStore() *URLMapStore {
	return &URLMapStore{
		store: sync.Map{},
	}
}

func (s *URLMapStore) SaveURL(url string) (string, error) {
	id := utils.NewRandomString(urlIDLength)
	s.store.Store(id, url)
	return id, nil
}

func (s *URLMapStore) GetURL(id string) (string, error) {
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
