package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/portbound/go-fs/internal/auth"
	"github.com/portbound/go-fs/internal/config"
	"github.com/portbound/go-fs/internal/handlers"
	"github.com/portbound/go-fs/internal/infrastructure/database/sqlite"
	"github.com/portbound/go-fs/internal/infrastructure/storage/gcs"
	"github.com/portbound/go-fs/internal/middleware"
	"github.com/portbound/go-fs/internal/repositories"
	"github.com/portbound/go-fs/internal/services"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	if err := os.MkdirAll(cfg.TmpDir, 0755); err != nil {
		log.Fatalf("failed to create tmp directory '%s': %v", cfg.TmpDir, err)
	}

	db, err := setupDB(cfg.DBEngine, cfg.DBConnStr)
	if err != nil {
		log.Fatalf("main.setupDB failed: %v", err)
	}
	defer db.Conn.Close()

	storageRepo, err := setupStorage(cfg.StorageProvider, cfg.BucketName)
	if err != nil {
		log.Fatalf("main.setupStorage failed: %v", err)
	}

	logFile, logger, err := setupLogging(cfg.LogDir)
	if err != nil {
		log.Fatalf("main.setupLogging failed: %v", err)
	}
	defer logFile.Close()

	authenticator := auth.NewAuthenticator(cfg.JWTSecret)
	fileMetaService := services.NewFileMetaService(db)
	userService := services.NewUserService(db)
	fileService := services.NewFileService(storageRepo, fileMetaService, logger, cfg.TmpDir)

	webHandler := handlers.NewWebHandler(authenticator, userService)
	apiHandler := handlers.NewAPIHandler(fileService, fileMetaService)

	mux := http.NewServeMux()
	webHandler.RegisterRoutes(mux)

	apiMux := http.NewServeMux()
	apiHandler.RegisterRoutes(apiMux)

	authMW := middleware.NewAuthMiddleware(authenticator, userService)
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

func setupStorage(storageProvider, bucket string) (repositories.StorageRepository, error) {
	switch storageProvider {
	case "gcs":
		ctx := context.Background()
		return gcs.NewStorage(ctx, bucket)
	default:
		return nil, fmt.Errorf("unsupported cloud provider: %s", storageProvider)
	}
}

func setupLogging(dir string) (*os.File, *log.Logger, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, nil, fmt.Errorf("failed to create log directory '%s': %w", dir, err)
	}

	logName := fmt.Sprintf("%s.log", time.Now().Format("2006-01-02"))
	logFilePath := filepath.Join(dir, logName)

	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open log file '%s': %w", logFilePath, err)
	}

	logger := log.New(logFile, "", log.Ldate|log.Ltime|log.Lshortfile)
	return logFile, logger, nil
}
