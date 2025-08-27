package repositories

import (
	"context"
	"io"
	"time"
)

type StorageRepository interface {
	Upload(ctx context.Context, name string, path string) (int64, time.Time, error)
	Download(ctx context.Context, fileName string) (io.ReadCloser, error)
	Delete(ctx context.Context, fileName string) error
}
