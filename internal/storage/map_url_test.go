package storage

import (
	"context"
	"os"
	"testing"
)

func BenchmarkSaveURL(b *testing.B) {
	ctx := context.Background()

	tmpFile, err := os.CreateTemp("./", "test_storage_*.json")
	if err != nil {
		b.Fatalf("failed to create temporary file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	store, err := NewURLMapStore(ctx, tmpFile.Name())
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

	store, err := NewURLMapStore(ctx, tmpFile.Name())
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

	store, err := NewURLMapStore(ctx, tmpFile.Name())
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

	store, err := NewURLMapStore(ctx, tmpFile.Name())
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
