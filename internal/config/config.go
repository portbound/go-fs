// Package config
package config

import (
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	ServerPort      string `envconfig:"SERVER_PORT" required:"true"`
	Environment     string `envconfig:"ENVIRONMENT" required:"true"`
	DBConnStr       string `envconfig:"DB_CONNECTION_STRING" required:"true"`
	DBEngine        string `envconfig:"DB_ENGINE" required:"true"`
	StorageProvider string `envconfig:"STORAGE_PROVIDER" required:"true"`
	TmpDir          string `envconfig:"TMP_DIR" default:"./local/tmp"`
	LogDir          string `envconfig:"LOG_DIR" default:"./local/logs"`
	GoogleClientID  string `envconfig:"GOOGLE_CLIENT_ID" required:"true"`
	ProjectID       string `envconfig:"GCS_PROJECT_ID" required:"true"`
	JWTSecret       string `envconfig:"JWT_SECRET" required:"true"`
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
