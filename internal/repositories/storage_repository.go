package repositories

import (
	"context"
	"io"

	"cloud.google.com/go/storage"
)

type StorageRepository interface {
	Upload(ctx context.Context, name string, path string) error
	Download(ctx context.Context, fileName string) (io.ReadCloser, error)
	ListObjects(ctx context.Context, query *storage.Query) ([]string, error)
	Delete(ctx context.Context, fileName string) error
}
