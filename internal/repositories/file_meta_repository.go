// Package repositories
package repositories

import (
	"context"

	"github.com/portbound/go-fs/internal/models"
)

type FileMetaRepository interface {
	CreateFileMeta(ctx context.Context, filemeta *models.FileMeta) error
	GetFileMeta(ctx context.Context, id string) (*models.FileMeta, error)
	GetAllFileMeta(ctx context.Context, owner *models.User) ([]*models.FileMeta, error)
	DeleteFileMeta(ctx context.Context, id string) error
}
