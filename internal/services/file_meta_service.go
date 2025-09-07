package services

import (
	"context"
	"fmt"

	"github.com/portbound/go-fs/internal/models"
	"github.com/portbound/go-fs/internal/repositories"
)

type FileMetaService interface {
	SaveFileMeta(ctx context.Context, fm *models.FileMeta) error
	LookupFileMeta(ctx context.Context, id string) (*models.FileMeta, error)
	LookupAllFileMeta(ctx context.Context, owner *models.User) ([]*models.FileMeta, error)
	DeleteFileMeta(ctx context.Context, id string) error
}

type fileMetaService struct {
	db repositories.FileMetaRepository
}

func NewFileMetaService(fileRepo repositories.FileMetaRepository) FileMetaService {
	return &fileMetaService{db: fileRepo}
}

func (fms *fileMetaService) SaveFileMeta(ctx context.Context, fm *models.FileMeta) error {
	if err := fms.db.CreateFileMeta(ctx, fm); err != nil {
		return fmt.Errorf("[services.SaveFileMeta] failed to save file metadata: %w", err)
	}
	return nil
}

func (fms *fileMetaService) LookupFileMeta(ctx context.Context, id string) (*models.FileMeta, error) {
	fm, err := fms.db.GetFileMeta(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("[services.LookupFileMeta] failed to get file for id '%s': %w", id, err)
	}
	return fm, nil
}

func (fms *fileMetaService) LookupAllFileMeta(ctx context.Context, owner *models.User) ([]*models.FileMeta, error) {
	data, err := fms.db.GetAllFileMeta(ctx, owner)
	if err != nil {
		return nil, fmt.Errorf("[services.GetFileIDs] failed to get file ids from DB: %w", err)
	}

	var fm []*models.FileMeta
	for _, item := range data {
		if item.ParentID == "" {
			fm = append(fm, item)
		}
	}
	return fm, nil
}

func (fms *fileMetaService) DeleteFileMeta(ctx context.Context, id string) error {
	if err := fms.db.DeleteFileMeta(ctx, id); err != nil {
		return fmt.Errorf("[services.DeleteFileMeta] failed to delete file metadata: %w", err)
	}
	return nil
}
