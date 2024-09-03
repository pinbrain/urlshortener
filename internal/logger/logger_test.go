package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitialize(t *testing.T) {
	err := Initialize("debug")
	if err != nil {
		t.Errorf("Expected no error init logger, got: %v", err)
	}

	err = Initialize("invalid")
	assert.Error(t, err)
}
