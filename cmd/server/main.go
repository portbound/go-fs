package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/portbound/go-fs/internal/api"
	"github.com/portbound/go-fs/internal/database/sqlite"
	"github.com/portbound/go-fs/internal/fs"
	"github.com/portbound/go-fs/internal/middleware"
	"github.com/portbound/go-fs/internal/repositories"
	"github.com/portbound/go-fs/internal/services"
	"github.com/portbound/go-fs/internal/storage/gcs"
	"github.com/portbound/go-fs/internal/user"
	"github.com/portbound/go-fs/pkg/auth"
)

func main() {
	cfg, err := Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	accessLog, err := os.OpenFile(filepath.Join(cfg.LogDir, "access.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal("failed to set up access log")
	}
	defer accessLog.Close()

	errorLog, err := os.OpenFile(filepath.Join(cfg.LogDir, "error.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal("failed to set up error log")
	}
	defer errorLog.Close()

	accessLogger := slog.New(slog.NewJSONHandler(accessLog, &slog.HandlerOptions{Level: slog.LevelInfo}))
	errorLogger := slog.New(slog.NewJSONHandler(errorLog, &slog.HandlerOptions{Level: slog.LevelInfo}))

	sqlite, err := sqlite.NewSQLiteDB(cfg.DBConnStr)
	if err != nil {
		log.Fatalf("main.setupDB failed: %v", err)
	}
	defer sqlite.Conn.Close()

	gcs, err := gcs.New(cfg.ProjectId)
	if err != nil {
		log.Fatalf("main.setupStorage failed: %v", err)
	}

	fsService := fs.NewService(sqlite, gcs, cfg.TmpDir)
	userService := user.NewService(sqlite)
	authenticator := auth.NewAuthenticator(cfg.JWTSecret, cfg.GoogleClientID, cfg.Environment)

	mux := http.NewServeMux()
	webHandler := handlers.NewWebHandler(authenticator, userService, errorLogger)
	webHandler.RegisterRoutes(mux)

	apiMux := http.NewServeMux()
	apiHandler := api.New(fsService, userService, errorLogger)
	apiHandler.RegisterRoutes(apiMux)

	authMW := middleware.NewAuthMiddleware(authenticator, userService)
	loggingMW := middleware.NewLoggingMiddleware(accessLogger)
	if cfg.Environment != "development" {
		mux.Handle("/", authMW.RequireWebAuth(http.FileServer(http.Dir("./web/public"))))
		mux.Handle("/api/", authMW.RequireAPIAuth(http.StripPrefix("/api", apiMux)))
	} else {
		mux.Handle("/", http.FileServer(http.Dir("./web/public")))
		mux.Handle("/api/", http.StripPrefix("/api", apiMux))
	}

	server := http.Server{
		Addr:    cfg.ServerPort,
		Handler: loggingMW.LogRequest(mux),
	}

	log.Printf("starting server on port %s\n", cfg.ServerPort)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("error: server failed to start: %v", err)
	}
}

func setupDB(driverName string, connStr string) (*sqlite.SQLiteDB, error) {
	switch driverName {
	case "sqlite3":
		return
	default:
		return nil, fmt.Errorf("unsupported database engine: %s", driverName)
	}
}

func setupStorage(storageProvider string, projectId string) (repositories.StorageRepository, error) {
	switch storageProvider {
	case "gcs":
		return gcs.NewStorage(ctx, projectId)
	default:
		return nil, fmt.Errorf("unsupported cloud provider: %s", storageProvider)
	}
}
