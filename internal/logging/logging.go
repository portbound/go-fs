package logging

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
)

const dir string = "./logs"

type Logger struct {
	writer *slog.Logger
}

func New() (*Logger, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}

	accessLog, err := os.OpenFile(filepath.Join(dir, "access.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return nil, fmt.Errorf("create access log: %w", err)
	}
	defer accessLog.Close()

	errorLog, err := os.OpenFile(filepath.Join(dir, "error.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return nil, fmt.Errorf("create error log: %w", err)
	}
	defer errorLog.Close()

	accessHandler := slog.NewJSONHandler(accessLog, &slog.HandlerOptions{Level: slog.LevelInfo})
	errorHandler := slog.NewJSONHandler(errorLog, &slog.HandlerOptions{Level: slog.LevelError})
	multiHandler := slog.NewMultiHandler(accessHandler, errorHandler)

	return &Logger{
		writer: slog.New(multiHandler),
	}, nil
}

func (l *Logger) Request(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		l.writer.Info(r.Method, "URI", r.RequestURI, "client", r.RemoteAddr)
	})
}

func (l *Logger) Error(msg string, err error) {
	l.writer.Error(msg, "error", err)
}
