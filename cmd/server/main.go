package main

import (
	"log"
	"net/http"

	"github.com/portbound/go-fs/internal/config"
	"github.com/portbound/go-fs/internal/fs"
	"github.com/portbound/go-fs/internal/middleware"
	"github.com/portbound/go-fs/internal/platform/database/sqlite"
	"github.com/portbound/go-fs/internal/platform/storage/gcs"
	"github.com/portbound/go-fs/internal/user"
	"github.com/portbound/go-fs/pkg/auth"
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

	authenticator := auth.NewAuthenticator(cfg.JWTSecret, cfg.GoogleClientID, cfg.Environment)

	userService := user.NewService(sqlite)
	fsService := fs.NewService(sqlite, gcs, cfg.TmpDir)

	userMux := http.NewServeMux()
	userHandler := user.NewHandler(authenticator, userService, logger)
	userHandler.RegisterRoutes(userMux)

	fsMux := http.NewServeMux()
	fsHandler := fs.NewHandler(fsService, logger)
	fsHandler.RegisterRoutes(fsMux)

	authMW := middleware.NewAuthMiddleware(authenticator, userService)
	loggingMW := middleware.New(logger)

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
		Handler: logger.Request(userMux),
	}

	log.Printf("starting server on port %s\n", cfg.ServerPort)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("error: server failed to start: %v", err)
	}
}
