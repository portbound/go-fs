package repositories

import (
	"context"
	"io"
	"time"
)

type StorageRepository interface {
	Upload(ctx context.Context, fileName string, bucket string, src io.Reader) (int64, time.Time, error)
	Download(ctx context.Context, fileName string, bucket string) (io.ReadCloser, error)
	Delete(ctx context.Context, fileName string, bucket string) error
}
