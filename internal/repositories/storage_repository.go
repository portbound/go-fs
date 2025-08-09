package repositories

import (
	"context"
	"io"

	"github.com/portbound/go-fs/internal/models"
)

type StorageRepository interface {
	Upload(ctx context.Context, path string) error
	Download(ctx context.Context, fm *models.FileMeta) (io.ReadCloser, error)
	Delete(ctx context.Context, fileName string) error
}
