package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort       string
	DatabaseURL      string
	DatabaseENG      string
	LocalStoragePath string
}

func Load() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		log.Printf("failed to load .env: %v", err)
	}

	cfg := Config{
		ServerPort:       os.Getenv("SERVER_PORT"),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		DatabaseENG:      os.Getenv("DATABASE_ENG"),
		LocalStoragePath: os.Getenv("LOCAL_STORAGE_PATH"),
	}

	if cfg.DatabaseURL == "" {
		cfg.DatabaseURL = "sqlite"
	}

	return &cfg, nil
}
