package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/portbound/go-fs/internal/config"
	"github.com/portbound/go-fs/internal/fs"
	"github.com/portbound/go-fs/internal/middleware"
	"github.com/portbound/go-fs/internal/platform/database/sqlite"
	"github.com/portbound/go-fs/internal/platform/storage/gcs"
	"github.com/portbound/go-fs/internal/user"
	"github.com/portbound/go-fs/pkg/auth"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	accessLog, err := os.OpenFile(filepath.Join(cfg.LogDir, "access.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal("create access log: %v", err)
	}
	defer accessLog.Close()

	errorLog, err := os.OpenFile(filepath.Join(cfg.LogDir, "error.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal("create error log: %v", err)
	}
	defer errorLog.Close()

	// TODO: I'm pretty sure slog supports multi-file writing now, so I can use the same logger to write to both the access and error log
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

	authenticator := auth.NewAuthenticator(cfg.JWTSecret, cfg.GoogleClientID, cfg.Environment)

	userService := user.NewService(sqlite)
	fsService := fs.NewService(sqlite, gcs, cfg.TmpDir)

	userMux := http.NewServeMux()
	userHandler := user.NewHandler(authenticator, userService, errorLogger)
	userHandler.RegisterRoutes(userMux)

	fsMux := http.NewServeMux()
	fsHandler := fs.NewHandler(fsService, userService, errorLogger)
	fsHandler.RegisterRoutes(fsMux)

	authMW := middleware.NewAuthMiddleware(authenticator, userService)
	loggingMW := middleware.NewLoggingMiddleware(accessLogger)

	switch cfg.Environment {
	case "development":
		userMux.Handle("/", http.FileServer(http.Dir("./web/public")))
		userMux.Handle("/api/", http.StripPrefix("/api", fsMux))
	default:
		userMux.Handle("/", authMW.RequireWebAuth(http.FileServer(http.Dir("./web/public"))))
		userMux.Handle("/api/", authMW.RequireAPIAuth(http.StripPrefix("/api", fsMux)))
	}

	server := http.Server{
		Addr:    cfg.ServerPort,
		Handler: loggingMW.LogRequest(userMux),
	}

	log.Printf("starting server on port %s\n", cfg.ServerPort)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("error: server failed to start: %v", err)
	}
}
