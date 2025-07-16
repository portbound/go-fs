package repositories

import (
	"github.com/google/uuid"
	"github.com/portbound/go-fs/internal/models"
)

type FileRepository interface {
	Create(file *models.File) error
	Get(id uuid.UUID) (*models.File, error)
	GetAll() ([]*models.File, error)
	Update(id uuid.UUID, file *models.File) error
	Delete(id uuid.UUID) error
}
