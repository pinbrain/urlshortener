package mocks

import "github.com/stretchr/testify/mock"

type MockURLStorage struct {
	mock.Mock
}

func (m *MockURLStorage) SaveURL(url string) (string, error) {
	args := m.Called(url)
	return args.String(0), args.Error(1)
}

func (m *MockURLStorage) GetURL(id string) (string, error) {
	args := m.Called(id)
	return args.String(0), args.Error(1)
}

func (m *MockURLStorage) IsValidID(id string) bool {
	args := m.Called(id)
	return args.Bool(0)
}
