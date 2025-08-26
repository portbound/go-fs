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

	fileRepo, err := buildFileRepo(cfg.DatabaseENG, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to build file repository: %v", err)
	}

	storageRepo, err := buildStorageRepo(cfg.StorageProvider, cfg.BucketName)
	if err != nil {
		log.Fatalf("failed to build storage repository: %v", err)
	}

	fileService, err := services.NewFileService(fileRepo, storageRepo, cfg.TmpDir, cfg.LogsDir)
	if err != nil {
		log.Fatalf("failed to build file service: %v", err)
	}
	defer fileService.CloseLog()

	webHandler := handlers.NewAPIHandler(fileService)

	mux := http.NewServeMux()
	webHandler.RegisterRoutes(mux)

	log.Printf("starting server on port %s\n", cfg.ServerPort)
	if err := http.ListenAndServe(cfg.ServerPort, mux); err != nil {
		log.Fatalf("error: server failed to start: %v", err)
	}
}

func buildFileRepo(dbEngine, dbURL string) (repositories.FileRepository, error) {
	switch dbEngine {
	case "sqlite":
		return sqlite.NewDB(dbURL)
	default:
		return nil, fmt.Errorf("unsupported database engine: %s", dbEngine)
	}
}

func buildStorageRepo(storageProvider, bucket string) (repositories.StorageRepository, error) {
	switch storageProvider {
	case "gcs":
		ctx := context.Background()
		return gcs.NewStorage(ctx, bucket)
	default:
		return nil, fmt.Errorf("unsupported cloud provider: %s", storageProvider)
	}
}
