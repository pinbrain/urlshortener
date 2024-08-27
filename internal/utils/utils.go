// Модуль utils предоставляет различные универсальные функции, используемые по всему приложению.
package utils

import (
	"math/rand"
	"net/url"
	"time"
)

// Набор символов, из которых формируется случайная строка.
const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// NewRandomString генерирует случайную строку из символов chars длиной length.
func NewRandomString(length int) string {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	randomBytes := make([]byte, length)
	for i := range randomBytes {
		randomBytes[i] = chars[rnd.Intn(len(chars))]
	}

	return string(randomBytes)
}

// IsValidURLString проверяет корректность сокращенной ссылки (проверка формата).
func IsValidURLString(urlStr string) bool {
	parsedURL, err := url.ParseRequestURI(urlStr)
	return err == nil && parsedURL.Scheme != "" && parsedURL.Host != ""
}
