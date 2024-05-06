package utils

import (
	"math/rand"
	"net/url"
	"time"
)

const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

func NewRandomString(length int) string {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	randomBytes := make([]byte, length)
	for i := range randomBytes {
		randomBytes[i] = chars[rnd.Intn(len(chars))]
	}

	return string(randomBytes)
}

func IsValidURLString(urlStr string) bool {
	parsedURL, err := url.ParseRequestURI(urlStr)
	return err == nil && parsedURL.Scheme != "" && parsedURL.Host != ""
}
