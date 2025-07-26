package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/portbound/go-fs/internal/config"
	"github.com/portbound/go-fs/internal/handlers"
	"github.com/portbound/go-fs/internal/infrastructure/database/sqlite"
	"github.com/portbound/go-fs/internal/infrastructure/storage/gcs"
	"github.com/portbound/go-fs/internal/repositories"
	"github.com/portbound/go-fs/internal/services"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	fileRepo, err := createFileRepo(cfg)
	if err != nil {
		log.Fatalf("failed to create file repository: %v", err)
	}

	storageRepo, err := createStorageRepo(cfg)
	if err != nil {
		log.Fatalf("failed to create storage repository: %v", err)
	}

	fileService := services.NewFileService(fileRepo, storageRepo, cfg.TmpDir)
	fileHandler := handlers.NewFileHandler(fileService)

	mux := http.NewServeMux()
	fileHandler.RegisterRoutes(mux)

	log.Printf("starting server on port %s\n", cfg.ServerPort)
	if err := http.ListenAndServe(cfg.ServerPort, mux); err != nil {
		log.Fatalf("error: server failed to start: %v", err)
	}
}

func createFileRepo(cfg *config.Config) (repositories.FileRepository, error) {
	switch cfg.DatabaseENG {
	case "sqlite":
		return sqlite.NewDB(cfg.DatabaseURL)
	default:
		return nil, fmt.Errorf("unsupported database engine: %s", cfg.DatabaseENG)
	}
}

func createStorageRepo(cfg *config.Config) (repositories.StorageRepository, error) {
	switch cfg.StorageProvider {
	case "gcs":
		return gcs.NewGCSStorage(cfg.BucketName)
	default:
		return nil, fmt.Errorf("unsupported cloud provider: %s", cfg.StorageProvider)
	}
}
