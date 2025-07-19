package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/portbound/go-fs/internal/models"
)

type FileRepository interface {
	Create(ctx context.Context, file *models.FileMetadata) error
	Get(ctx context.Context, id uuid.UUID) (*models.FileMetadata, error)
	GetAll(ctx context.Context) ([]*models.FileMetadata, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
