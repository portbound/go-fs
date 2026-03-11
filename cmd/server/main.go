package main

import (
	"log"
	"net/http"

	"github.com/portbound/go-fs/internal/auth"
	"github.com/portbound/go-fs/internal/config"
	"github.com/portbound/go-fs/internal/fs"
	"github.com/portbound/go-fs/internal/platform/database/sqlite"
	"github.com/portbound/go-fs/internal/platform/storage/gcs"
	"github.com/portbound/go-fs/internal/user"
	"github.com/portbound/portlog"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	logger, err := portlog.New()
	if err != nil {
		log.Fatalf("set up logging: %v", err)
	}
	defer logger.Close()

	sqlite, err := sqlite.NewSQLiteDB(cfg.DBConnectionString)
	if err != nil {
		log.Fatalf("set up database: %v", err)
	}
	defer sqlite.Conn.Close()

	gcs, err := gcs.New(cfg.GCSProjectId)
	if err != nil {
		log.Fatalf("set up storage: %v", err)
	}

	authenticator := auth.New(cfg.JWTSecret, cfg.GoogleClientID, cfg.Environment)
	userProvider := user.NewService(sqlite)
	authService := auth.NewService(authenticator, userProvider)
	authHandler := auth.NewHandler(authService, logger)

	fsService := fs.NewService(sqlite, gcs)
	fsHandler := fs.NewHandler(fsService, logger)

	authMux := http.NewServeMux()
	authHandler.RegisterRoutes(authMux)

	fsMux := http.NewServeMux()
	fsHandler.RegisterRoutes(fsMux)

	switch cfg.Environment {
	case "development":
		authMux.Handle("/", http.FileServer(http.Dir("./web/public")))
		authMux.Handle("/api/", http.StripPrefix("/api", fsMux))
	default:
		authMux.Handle("/", authHandler.RequireWebAuth(http.FileServer(http.Dir("./web/public"))))
		authMux.Handle("/api/", authHandler.RequireAPIAuth(http.StripPrefix("/api", fsMux)))
	}

	server := http.Server{
		Addr:    cfg.ServerPort,
		Handler: logger.Request(authMux),
	}

	log.Printf("starting server on port %s\n", cfg.ServerPort)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("error: server failed to start: %v", err)
	}
}
