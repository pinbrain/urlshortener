package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type MockURLStorage struct {
	mock.Mock
}

func (m *MockURLStorage) SaveURL(_ context.Context, url string) (string, error) {
	args := m.Called(url)
	return args.String(0), args.Error(1)
}

func (m *MockURLStorage) GetURL(_ context.Context, id string) (string, error) {
	args := m.Called(id)
	return args.String(0), args.Error(1)
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
