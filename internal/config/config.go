// Package config
package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort      string
	DBConnStr       string
	DBEngine        string
	StorageProvider string
	BucketName      string
	TmpDir          string
	LogDir          string
	GoogleClientID  string
	JWTSecret       string
}

func Load() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		return nil, err
	}

	cfg := Config{
		ServerPort:      os.Getenv("SERVER_PORT"),
		DBConnStr:       os.Getenv("DB_CONNECTION_STRING"),
		DBEngine:        os.Getenv("DB_ENGINE"),
		StorageProvider: os.Getenv("STORAGE_PROVIDER"),
		BucketName:      os.Getenv("BUCKET_NAME"),
		TmpDir:          os.Getenv("TMP_DIR"),
		LogDir:          os.Getenv("LOG_DIR"),
		GoogleClientID:  os.Getenv("GOOGLE_CLIENT_ID"),
		JWTSecret:       os.Getenv("JWT_SECRET"),
	}

	if cfg.ServerPort == "" {
		return nil, fmt.Errorf("SERVER_PORT is required but was undefined")
	}

	if cfg.DBConnStr == "" {
		return nil, fmt.Errorf("DATABASE_URL is required but was undefined")
	}

	if cfg.DBEngine == "" {
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

	if cfg.LogDir == "" {
		cfg.LogDir = "./local/logs"
		os.Mkdir(cfg.LogDir, 0755)
	}

	if cfg.GoogleClientID == "" {
		return nil, fmt.Errorf("GOOGLE_CLIENT_ID is required but was undefined")
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required but was undefined")
	}

	return &cfg, nil
}
