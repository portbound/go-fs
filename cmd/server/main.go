package main

import (
	"context"
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

	fileRepo, err := buildFileRepo(cfg)
	if err != nil {
		log.Fatalf("failed to build file repository: %v", err)
	}

	storageRepo, err := buildStorageRepo(cfg)
	if err != nil {
		log.Fatalf("failed to build storage repository: %v", err)
	}

	fileService, err := services.NewFileService(fileRepo, storageRepo, cfg.TmpDir, cfg.LogsDir)
	if err != nil {
		log.Fatalf("failed to build file service: %v", err)
	}
	defer fileService.CloseLog()

	apiHandler := handlers.NewAPIHandler(fileService)
	pageHandler := handlers.NewPageHandler(fileService)

	mux := http.NewServeMux()
	apiHandler.RegisterRoutes(mux)
	pageHandler.RegisterRoutes(mux)

	log.Printf("starting server on port %s\n", cfg.ServerPort)
	if err := http.ListenAndServe(cfg.ServerPort, mux); err != nil {
		log.Fatalf("error: server failed to start: %v", err)
	}
}

func buildFileRepo(cfg *config.Config) (repositories.FileRepository, error) {
	switch cfg.DatabaseENG {
	case "sqlite":
		return sqlite.NewDB(cfg.DatabaseURL)
	default:
		return nil, fmt.Errorf("unsupported database engine: %s", cfg.DatabaseENG)
	}
}

func buildStorageRepo(cfg *config.Config) (repositories.StorageRepository, error) {
	switch cfg.StorageProvider {
	case "gcs":
		ctx := context.Background()
		return gcs.NewStorage(ctx, cfg.BucketName)
	default:
		return nil, fmt.Errorf("unsupported cloud provider: %s", cfg.StorageProvider)
	}
}
