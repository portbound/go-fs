// Package config
package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort      string
	DatabaseURL     string
	DatabaseENG     string
	StorageProvider string
	BucketName      string
	TmpDir          string
	LogsDir         string
}

func Load() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		return nil, err
	}

	cfg := Config{
		ServerPort:      os.Getenv("SERVER_PORT"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		DatabaseENG:     os.Getenv("DATABASE_ENG"),
		StorageProvider: os.Getenv("STORAGE_PROVIDER"),
		BucketName:      os.Getenv("BUCKET_NAME"),
		TmpDir:          os.Getenv("TMP_DIR"),
		LogsDir:         os.Getenv("LOG_DIR"),
	}

	if cfg.ServerPort == "" {
		return nil, fmt.Errorf("SERVER_PORT is required but was undefined")
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required but was undefined")
	}

	if cfg.DatabaseENG == "" {
		return nil, fmt.Errorf("DATABASE_ENG is required but was undefined")
	}

	if cfg.StorageProvider == "" {
		return nil, fmt.Errorf("STORAGE_PROVIDER is required but was undefined")
	}

	if cfg.BucketName == "" {
		return nil, fmt.Errorf("BUCKET_NAME is required but was undefined")
	}

	if cfg.TmpDir == "" {
		cfg.TmpDir = "./local/tmp"
		os.Mkdir(cfg.TmpDir, 0755)
	}

	if cfg.LogsDir == "" {
		cfg.LogsDir = "./local/logs"
		os.Mkdir(cfg.LogsDir, 0755)
	}

	return &cfg, nil
}
