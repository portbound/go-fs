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
	"github.com/portbound/go-fs/internal/middleware"
	"github.com/portbound/go-fs/internal/repositories"
	"github.com/portbound/go-fs/internal/services"
	"github.com/portbound/go-fs/pkg/auth"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := setupDB(cfg.DBEngine, cfg.DBConnStr)
	if err != nil {
		log.Fatalf("main.setupDB failed: %v", err)
	}
	defer db.Conn.Close()

	storage, err := setupStorage(cfg.StorageProvider, cfg.BucketName)
	if err != nil {
		log.Fatalf("main.setupStorage failed: %v", err)
	}

	fileMetaService := services.NewFileMetaService(db)
	fileService := services.NewFileService(storage, fileMetaService, cfg.TmpDir)
	userService := services.NewUserService(db)

	authenticator := auth.NewAuthenticator(cfg.JWTSecret, cfg.GoogleClientID)
	authMW := middleware.NewAuthMiddleware(authenticator, userService)

	mux := http.NewServeMux()
	webHandler := handlers.NewWebHandler(authenticator, userService)
	webHandler.RegisterRoutes(mux)

	apiMux := http.NewServeMux()
	apiHandler := handlers.NewAPIHandler(fileService, fileMetaService)
	apiHandler.RegisterRoutes(apiMux)

	mux.Handle("/", authMW.RequireWebAuth(http.FileServer(http.Dir("./web/public"))))
	mux.Handle("/api/", authMW.RequireAPIAuth(http.StripPrefix("/api", apiMux)))

	server := http.Server{
		Addr:    cfg.ServerPort,
		Handler: mux,
	}

	log.Printf("starting server on port %s\n", cfg.ServerPort)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("error: server failed to start: %v", err)
	}
}

func setupDB(driverName string, connStr string) (*sqlite.SQLiteDB, error) {
	switch driverName {
	case "sqlite3":
		return sqlite.NewSQLiteDB(connStr)
	default:
		return nil, fmt.Errorf("unsupported database engine: %s", driverName)
	}
}

func setupStorage(storageProvider string, bucket string) (repositories.StorageRepository, error) {
	switch storageProvider {
	case "gcs":
		ctx := context.Background()
		return gcs.NewStorage(ctx, bucket)
	default:
		return nil, fmt.Errorf("unsupported cloud provider: %s", storageProvider)
	}
}
