package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/portbound/go-fs/internal/models"
)

type FileRepository interface {
	Create(ctx context.Context, file *models.File) error
	Get(ctx context.Context, id uuid.UUID) (*models.File, error)
	GetAll(ctx context.Context) ([]*models.File, error)
	Update(ctx context.Context, id uuid.UUID, file *models.File) error
	Delete(ctx context.Context, id uuid.UUID) error
}
