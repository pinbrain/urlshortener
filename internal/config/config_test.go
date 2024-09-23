package config

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestLoadJSON(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "config_*.json")
	if err != nil {
		t.Fatalf("Unable to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	jsonConfig := `{
		"server_address": ":6060",
		"base_url": "http://json.com",
		"file_storage_path": "/tmp/json.json",
		"database_dsn": "json_dsn",
		"enable_https": true
	}`
	_, err = tmpFile.Write([]byte(jsonConfig))
	if err != nil {
		t.Fatalf("Unable to write to temp file: %v", err)
	}
	tmpFile.Close()

	var cfg ServerConf
	cfg.JSONConfig = tmpFile.Name()

	err = loadJSON(&cfg)
	require.NoError(t, err)

	if cfg.ServerAddress != ":6060" {
		t.Errorf("Expected address :6060, got %s", cfg.ServerAddress)
	}
	if cfg.BaseURL.String() != "http://json.com" {
		t.Errorf("Expected base URL http://json.com, got %s", cfg.BaseURL.String())
	}
	if cfg.StorageFile != "/tmp/json.json" {
		t.Errorf("Expected storage file /tmp/json.json, got %s", cfg.StorageFile)
	}
	if cfg.DSN != "json_dsn" {
		t.Errorf("Expected DSN json_dsn, got %s", cfg.DSN)
	}
	if !cfg.EnableHTTPS {
		t.Errorf("Expected HTTPS to be enabled, but it wasn't")
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
