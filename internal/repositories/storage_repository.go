package repositories

import (
	"context"

	"github.com/portbound/go-fs/internal/models"
)

type StorageRepository interface {
	Upload(ctx context.Context, fm *models.FileMeta) error
	Delete(ctx context.Context, fm *models.FileMeta) error
}
