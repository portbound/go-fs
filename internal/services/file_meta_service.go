package services

import (
	"context"
	"fmt"

	"github.com/portbound/go-fs/internal/models"
	"github.com/portbound/go-fs/internal/repositories"
)

type FileMetaService struct {
	db repositories.FileRepository
}

func NewFileMetaService(fileRepo repositories.FileRepository) *FileMetaService {
	return &FileMetaService{db: fileRepo}
}

func (fms *FileMetaService) SaveFileMeta(ctx context.Context, fm *models.FileMeta) error {
	if err := fms.db.Create(ctx, fm); err != nil {
		return fmt.Errorf("services.SaveFileMeta: failed to save file metadata: %w", err)
	}
	return nil
}

func (fms *FileMetaService) LookupFileMeta(ctx context.Context, id string) (*models.FileMeta, error) {
	fm, err := fms.db.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("services.LookupFileMeta: failed to get file for id '%s': %w", id, err)
	}
	return fm, nil
}

func (fms *FileMetaService) LookupAllFileMeta(ctx context.Context) ([]*models.FileMeta, error) {
	data, err := fms.db.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("services.GetFileIDs: failed to get file ids from DB: %w", err)
	}

	var fm []*models.FileMeta
	for _, item := range data {
		if item.ParentID == "" {
			fm = append(fm, item)
		}
	}
	return fm, nil
}

func (fms *FileMetaService) DeleteFileMeta(ctx context.Context, id string) error {
	if err := fms.db.Delete(ctx, id); err != nil {
		return fmt.Errorf("services.DeleteFileMeta: failed to delete file metadata: %w", err)
	}
	return nil
}
