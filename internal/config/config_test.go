package config

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateBaseURL(t *testing.T) {
	_, err := validateBaseURL("http://valid.ru")
	if err != nil {
		t.Errorf("Expected no error validating url, got: %v", err)
	}

	_, err = validateBaseURL("!not_valid!")
	assert.Error(t, err)
}

func TestValidateStorageFileName(t *testing.T) {
	err := validateFileName("valid.json")
	if err != nil {
		t.Errorf("Expected no error validating file name, got: %v", err)
	}

	err = validateFileName("..")
	assert.Error(t, err)
}

func TestLoadFlags(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	oldFlagSet := flag.CommandLine
	flag.CommandLine = fs
	defer func() { flag.CommandLine = oldFlagSet }()

	os.Args = []string{
		"test",
		"-a", ":9090",
		"-l", "debug",
		"-d", "user:password@/dbname",
		"-f", "/var/tmp/short-url-db.json",
		"-b", "http://example.com:9090",
	}

	cfg := ServerConf{}
	err := loadFlags(&cfg)
	if err != nil {
		t.Errorf("Expected no error loading flags, got: %v", err)
	}

	expectedAddress := ":9090"
	if cfg.ServerAddress != expectedAddress {
		t.Errorf("Expected %v, got %v", expectedAddress, cfg.ServerAddress)
	}

	expectedLogLevel := "debug"
	if cfg.LogLevel != expectedLogLevel {
		t.Errorf("Expected %v, got %v", expectedLogLevel, cfg.LogLevel)
	}

	expectedStorageFile := "/var/tmp/short-url-db.json"
	if cfg.StorageFile != expectedStorageFile {
		t.Errorf("Expected %v, got %v", expectedStorageFile, cfg.StorageFile)
	}

	expectedBaseURL := "http://example.com:9090"
	if cfg.BaseURL.String() != expectedBaseURL {
		t.Errorf("Expected %v, got %v", expectedBaseURL, cfg.BaseURL.String())
	}

	expectedDSN := "user:password@/dbname"
	if cfg.DSN != expectedDSN {
		t.Errorf("Expected %v, got %v", expectedDSN, cfg.DSN)
	}
}

func TestLoadEnvs(t *testing.T) {
	t.Setenv("SERVER_ADDRESS", ":9090")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("FILE_STORAGE_PATH", "/var/tmp/short-url-db.json")
	t.Setenv("BASE_URL", "http://example.com:9090")
	t.Setenv("DATABASE_DSN", "user:password@/dbname")

	cfg := ServerConf{}
	err := loadEnvs(&cfg)
	if err != nil {
		t.Errorf("Expected no error loading environment variables, got: %v", err)
	}

	expectedAddress := ":9090"
	if cfg.ServerAddress != expectedAddress {
		t.Errorf("Expected %v, got %v", expectedAddress, cfg.ServerAddress)
	}

	expectedLogLevel := "debug"
	if cfg.LogLevel != expectedLogLevel {
		t.Errorf("Expected %v, got %v", expectedLogLevel, cfg.LogLevel)
	}

	expectedStorageFile := "/var/tmp/short-url-db.json"
	if cfg.StorageFile != expectedStorageFile {
		t.Errorf("Expected %v, got %v", expectedStorageFile, cfg.StorageFile)
	}

	expectedBaseURL := "http://example.com:9090"
	if cfg.BaseURL.String() != expectedBaseURL {
		t.Errorf("Expected %v, got %v", expectedBaseURL, cfg.BaseURL.String())
	}

	expectedDSN := "user:password@/dbname"
	if cfg.DSN != expectedDSN {
		t.Errorf("Expected %v, got %v", expectedDSN, cfg.DSN)
	}
}

func TestInitConfig(t *testing.T) {
	t.Setenv("SERVER_ADDRESS", ":9090")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("FILE_STORAGE_PATH", "/var/tmp/short-url-db.json")
	t.Setenv("BASE_URL", "http://example.com:9090")

	cfg, err := InitConfig()
	if err != nil {
		t.Errorf("Expected no error initializing config, got: %v", err)
	}

	expectedAddress := ":9090"
	if cfg.ServerAddress != expectedAddress {
		t.Errorf("Expected %v, got %v", expectedAddress, cfg.ServerAddress)
	}

	expectedLogLevel := "debug"
	if cfg.LogLevel != expectedLogLevel {
		t.Errorf("Expected %v, got %v", expectedLogLevel, cfg.LogLevel)
	}

	expectedStorageFile := "/var/tmp/short-url-db.json"
	if cfg.StorageFile != expectedStorageFile {
		t.Errorf("Expected %v, got %v", expectedStorageFile, cfg.StorageFile)
	}

	expectedBaseURL := "http://example.com:9090"
	if cfg.BaseURL.String() != expectedBaseURL {
		t.Errorf("Expected %v, got %v", expectedBaseURL, cfg.BaseURL.String())
	}
}
