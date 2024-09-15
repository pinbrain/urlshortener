package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetURL(t *testing.T) {
	ctx := context.Background()
	store, err := NewURLMapStore("")
	require.NoError(t, err)
	defer store.Close()

	tests := []struct {
		name      string
		url       string
		short     string
		isDeleted bool
		err       error
	}{
		{
			name:      "получена валидная ссылка",
			url:       "http://some.ru",
			isDeleted: false,
			err:       nil,
		},
		{
			name:      "ссылка удалена",
			url:       "http://some.ru",
			isDeleted: true,
			err:       ErrIsDeleted,
		},
		{
			name:      "ссылка не найдена",
			short:     "not_existing",
			isDeleted: true,
			err:       nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			short := tt.short
			if tt.url != "" {
				short, err = store.SaveURL(ctx, tt.url, 1)
				require.NoError(t, err)
			}
			if tt.isDeleted {
				err = store.DeleteUserURLs(1, []string{short})
				require.NoError(t, err)
			}
			full, err := store.GetURL(ctx, short)
			if tt.err == nil {
				require.NoError(t, err)
				assert.Equal(t, tt.url, full)
			} else {
				assert.Equal(t, tt.err, err)
			}
		})
	}
}

func TestGetUser(t *testing.T) {
	ctx := context.Background()
	store, err := NewURLMapStore("")
	require.NoError(t, err)
	defer store.Close()

	tests := []struct {
		name   string
		userID int
		create bool
		err    error
	}{
		{
			name:   "Данные пользователя получены",
			create: true,
			err:    nil,
		},
		{
			name:   "Невалидный ID пользователя",
			userID: -1,
			create: false,
			err:    errors.New("invalid user id"),
		},
		{
			name:   "Пользователь не найден",
			userID: 2,
			create: false,
			err:    ErrNoData,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID := tt.userID
			if tt.create {
				user, err := store.CreateUser(ctx)
				require.NoError(t, err)
				userID = user.ID
			}
			_, err = store.GetUser(ctx, userID)
			if tt.err == nil {
				require.NoError(t, err)
			} else {
				assert.Equal(t, tt.err, err)
			}
		})
	}
}

func TestGetUserURLs(t *testing.T) {
	ctx := context.Background()
	store, err := NewURLMapStore("")
	require.NoError(t, err)
	defer store.Close()

	urls := []ShortenURL{
		{
			Original: "http://some1.ru",
		},
		{
			Original: "http://some2.ru",
		},
	}

	err = store.SaveBatchURL(ctx, urls, 1)
	require.NoError(t, err)
	userURLs, err := store.GetUserURLs(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, urls, userURLs)
}

func TestProcessSyncFileData(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "jsonDB_*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	store := &URLMapStore{
		store: map[string]URLMapData{
			"short1": {OriginalURL: "http://example1.com", UserID: 1, IsDeleted: false},
			"short2": {OriginalURL: "http://example2.com", UserID: 2, IsDeleted: false},
		},
		jsonDB: jsonDB{
			file:    tmpFile,
			encoder: json.NewEncoder(tmpFile),
			decoder: json.NewDecoder(tmpFile),
		},
		mutex: sync.RWMutex{},
	}

	err = store.processSyncFileData()
	require.NoError(t, err)

	fileData, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)
	require.Contains(t, string(fileData), `"original_url":"http://example1.com","short_url":"short1"`)
	require.Contains(t, string(fileData), `"original_url":"http://example2.com","short_url":"short2"`)

	store.jsonDB.file.Close()
	require.NoError(t, os.Remove(tmpFile.Name()))
	err = store.processSyncFileData()
	require.Error(t, err)
}

func TestNewURLMapStore(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "jsonDB_*.json")
	fmt.Println(tmpFile.Name())
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	records := []URLMapFileRecord{
		{
			OriginalURL: "http://example1.com",
			ShortURL:    "short1",
			UserID:      1,
			IsDeleted:   false,
		},
		{
			OriginalURL: "http://example2.com",
			ShortURL:    "short2",
			UserID:      2,
			IsDeleted:   false,
		},
	}
	for _, record := range records {
		err = json.NewEncoder(tmpFile).Encode(record)
		require.NoError(t, err)
	}

	require.NoError(t, tmpFile.Close())

	store, err := NewURLMapStore(tmpFile.Name())
	require.NoError(t, err)
	defer require.NoError(t, store.Close())

	require.Len(t, store.store, 2)
	require.Equal(t, "http://example1.com", store.store["short1"].OriginalURL)
	require.Equal(t, "http://example2.com", store.store["short2"].OriginalURL)

	require.Len(t, store.userStore[1], 1)
	require.Len(t, store.userStore[2], 1)
	require.Contains(t, store.userStore[1], "short1")
	require.Contains(t, store.userStore[2], "short2")

	require.Equal(t, 2, store.userMaxID)

	require.NoError(t, os.Remove(tmpFile.Name()))
}

func BenchmarkSaveURL(b *testing.B) {
	ctx := context.Background()

	tmpFile, err := os.CreateTemp("./", "test_storage_*.json")
	if err != nil {
		b.Fatalf("failed to create temporary file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	store, err := NewURLMapStore(tmpFile.Name())
	if err != nil {
		b.Fatalf("failed to create store: %v", err)
	}

	url := "https://example.com"
	userID := 1

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = store.SaveURL(ctx, url, userID)
		if err != nil {
			b.Fatalf("failed to save URL: %v", err)
		}
	}
}

func BenchmarkSaveBatchURL(b *testing.B) {
	ctx := context.Background()

	tmpFile, err := os.CreateTemp("./", "test_storage_*.json")
	if err != nil {
		b.Fatalf("failed to create temporary file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	store, err := NewURLMapStore(tmpFile.Name())
	if err != nil {
		b.Fatalf("failed to create store: %v", err)
	}

	userID := 1
	urls := []ShortenURL{
		{Original: "https://example1.com"},
		{Original: "https://example2.com"},
		{Original: "https://example3.com"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = store.SaveBatchURL(ctx, urls, userID)
		if err != nil {
			b.Fatalf("failed to save batch URLs: %v", err)
		}
	}
}

func BenchmarkGetURL(b *testing.B) {
	ctx := context.Background()

	tmpFile, err := os.CreateTemp("./", "test_storage_*.json")
	if err != nil {
		b.Fatalf("failed to create temporary file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	store, err := NewURLMapStore(tmpFile.Name())
	if err != nil {
		b.Fatalf("failed to create store: %v", err)
	}

	url := "https://example.com"
	userID := 1
	shortURL, err := store.SaveURL(ctx, url, userID)
	if err != nil {
		b.Fatalf("failed to save URL: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = store.GetURL(ctx, shortURL)
		if err != nil || url == "" {
			b.Fatalf("failed to get URL: %v", err)
		}
	}
}

func BenchmarkDeleteUserURLs(b *testing.B) {
	ctx := context.Background()

	urlsCount := 2

	tmpFile, err := os.CreateTemp("./", "test_storage_*.json")
	if err != nil {
		b.Fatalf("failed to create temporary file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	store, err := NewURLMapStore(tmpFile.Name())
	if err != nil {
		b.Fatalf("failed to create store: %v", err)
	}

	userID := 1
	urls := []string{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		for j := 0; j < urlsCount; j++ {
			shortURL, saveErr := store.SaveURL(ctx, "https://example.com", userID)
			if saveErr != nil {
				b.Fatalf("failed to save URL: %v", err)
			}
			urls = append(urls, shortURL)
		}
		b.StartTimer()

		err = store.DeleteUserURLs(userID, urls)
		if err != nil {
			b.Fatalf("failed to delete URLs: %v", err)
		}
	}
}
