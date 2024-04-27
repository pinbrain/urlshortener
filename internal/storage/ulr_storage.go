package storage

import (
	"fmt"
	"regexp"

	"github.com/pinbrain/urlshortener/internal/utils"
)

type UrlStorage interface {
	SaveUrl(url string) (id string, err error)
	GetUrl(id string) (url string, err error)
	IsValidUrlID(id string) bool
}

const urlIdLength = 8

type UrlMapStore struct {
	store map[string]string
}

func NewUrlMapStore() *UrlMapStore {
	return &UrlMapStore{
		store: make(map[string]string),
	}
}

func (s *UrlMapStore) SaveUrl(url string) (id string, err error) {
	id = utils.NewRandomString(urlIdLength)
	s.store[id] = url
	return id, nil
}

func (s *UrlMapStore) GetUrl(id string) (url string, err error) {
	url, ok := s.store[id]
	if !ok {
		return "", nil
	}
	return url, nil
}

func (s *UrlMapStore) IsValidUrlID(id string) bool {
	regStr := fmt.Sprintf(`^[a-zA-Z0-9]{%d}$`, urlIdLength)
	validIDReg := regexp.MustCompile(regStr)
	return validIDReg.MatchString(id)
}
