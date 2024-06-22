package mocks

import (
	"context"

	"github.com/pinbrain/urlshortener/internal/storage"
	"github.com/stretchr/testify/mock"
)

type MockURLStorage struct {
	mock.Mock
}

func (m *MockURLStorage) SaveURL(_ context.Context, url string, _ int) (string, error) {
	args := m.Called(url)
	return args.String(0), args.Error(1)
}

func (m *MockURLStorage) GetURL(_ context.Context, id string) (string, error) {
	args := m.Called(id)
	return args.String(0), args.Error(1)
}

func (m *MockURLStorage) CreateUser(_ context.Context) (*storage.User, error) {
	return &storage.User{}, nil
}

func (m *MockURLStorage) GetUser(_ context.Context, id int) (*storage.User, error) {
	return &storage.User{ID: id}, nil
}

func (m *MockURLStorage) GetUserURLs(_ context.Context, _ int) ([]storage.ShortenURL, error) {
	return nil, nil
}

func (m *MockURLStorage) IsValidID(id string) bool {
	args := m.Called(id)
	return args.Bool(0)
}

func (m *MockURLStorage) Close() error {
	return nil
}

func (m *MockURLStorage) Ping(_ context.Context) error {
	return nil
}

func (m *MockURLStorage) SaveBatchURL(_ context.Context, _ []storage.ShortenURL, _ int) error {
	return nil
}
