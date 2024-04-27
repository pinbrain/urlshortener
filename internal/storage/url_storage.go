package storage

import (
	"fmt"
	"regexp"

	"github.com/pinbrain/urlshortener/internal/utils"
)

type URLStorage interface {
	SaveURL(url string) (id string, err error)
	GetURL(id string) (url string, err error)
	IsValidID(id string) bool
}

const urlIDLength = 8

type URLMapStore struct {
	store map[string]string
}

func NewURLMapStore() *URLMapStore {
	return &URLMapStore{
		store: make(map[string]string),
	}
}

func (s *URLMapStore) SaveURL(url string) (string, error) {
	id := utils.NewRandomString(urlIDLength)
	s.store[id] = url
	return id, nil
}

func (s *URLMapStore) GetURL(id string) (string, error) {
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
