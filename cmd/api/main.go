package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"cloud.google.com/go/storage"
	"github.com/portbound/go-fs/internal/config"
	"github.com/portbound/go-fs/internal/handlers"
	"github.com/portbound/go-fs/internal/infrastructure/database/sqlite"
	"github.com/portbound/go-fs/internal/repositories"
	"github.com/portbound/go-fs/internal/services"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	repo, err := createRepository(cfg)
	if err != nil {
		log.Fatalf("failed to create repository: %v", err)
	}

	client, err := createClient()
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	// ctx := context.Background()
	// it := client.Bucket("gofs_bucket_1").Objects(ctx, nil)
	//
	// for {
	// 	objAttrs, err := it.Next()
	// 	if err != nil {
	// 		if err == iterator.Done {
	// 			break
	// 		}
	// 		log.Fatalf("failed to iterate objects: %v", err)
	// 	}
	//
	// 	fmt.Printf("Object: %s (Size: %d bytes, Content-Type: %s)\n", objAttrs.Name, objAttrs.Size, objAttrs.ContentType)
	// }

	fileService := services.NewFileService(repo, cfg.LocalStoragePath, client)
	fileHandler := handlers.NewFileHandler(fileService)

	mux := http.NewServeMux()
	fileHandler.RegisterRoutes(mux)

	log.Printf("starting server on port %s\n", cfg.ServerPort)
	if err := http.ListenAndServe(cfg.ServerPort, mux); err != nil {
		log.Fatalf("error: server failed to start: %v", err)
	}
}

func createRepository(cfg *config.Config) (repositories.FileRepository, error) {
	switch cfg.DatabaseENG {
	case "sqlite":
		return sqlite.NewDB(cfg.DatabaseURL)
	default:
		return nil, fmt.Errorf("unsupported database engine: %s", cfg.DatabaseENG)
	}
}

func createClient() (*storage.Client, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return client, nil
}
