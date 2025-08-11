// Package repositories
package repositories

import (
	"context"

	"github.com/portbound/go-fs/internal/models"
)

type FileRepository interface {
	Create(ctx context.Context, filemeta *models.FileMeta) error
	Get(ctx context.Context, id string) (*models.FileMeta, error)
	GetAll(ctx context.Context) ([]*models.FileMeta, error)
	Delete(ctx context.Context, id string) error
}
