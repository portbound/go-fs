package repositories

import (
	"context"
	"io"
	"time"

	"cloud.google.com/go/storage"
)

type StorageRepository interface {
	Upload(ctx context.Context, name string, path string) (int64, time.Time, error)
	Download(ctx context.Context, fileName string) (io.ReadCloser, error)
	ListObjects(ctx context.Context, query *storage.Query) ([]string, error)
	Delete(ctx context.Context, fileName string) error
}
