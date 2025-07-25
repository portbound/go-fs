package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort       string
	DatabaseURL      string
	DatabaseENG      string
	LocalStoragePath string
	CloudKey         string
}

func Load() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		return nil, err
	}

	cfg := Config{
		ServerPort:       os.Getenv("SERVER_PORT"),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		DatabaseENG:      os.Getenv("DATABASE_ENG"),
		LocalStoragePath: os.Getenv("LOCAL_STORAGE_PATH"),
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

	if cfg.LocalStoragePath == "" {
		return nil, fmt.Errorf("LOCAL_STORAGE_PATH is required but was undefined")
	}

	return &cfg, nil
}
