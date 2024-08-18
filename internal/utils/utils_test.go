package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRandomString(t *testing.T) {
	randString := NewRandomString(10)
	assert.Len(t, randString, 10)
	anotherString := NewRandomString(10)
	assert.NotEqual(t, randString, anotherString)
}

func BenchmarkNewRandomString(b *testing.B) {
	length := 10
	for i := 0; i < b.N; i++ {
		NewRandomString(length)
	}
}
