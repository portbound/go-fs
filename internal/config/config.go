package config

import (
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	ServerPort         string `envconfig:"SERVER_PORT" required:"true"`
	Environment        string `envconfig:"ENVIRONMENT" required:"true"`
	DBConnectionString string `envconfig:"DB_CONNECTION_STRING" default:"data/sqlite.db" required:"true"`
	// GOOGLE_APPLICATION_CREDENTIALS
	GoogleClientID string `envconfig:"GOOGLE_CLIENT_ID" required:"true"`
	GCSProjectId   string `envconfig:"GCS_PROJECT_ID" required:"true"`
	JWTSecret      string `envconfig:"JWT_SECRET" required:"true"`
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
