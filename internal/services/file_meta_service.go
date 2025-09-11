package services

import (
	"context"
	"fmt"
	"time"

	"github.com/portbound/go-fs/internal/models"
	"github.com/portbound/go-fs/internal/repositories"
)

type FileMetaService interface {
	SaveFileMeta(ctx context.Context, fm *models.FileMeta) error
	LookupFileMeta(ctx context.Context, id string, owner *models.User) (*models.FileMeta, error)
	LookupFileMetaByNameAndOwner(ctx context.Context, name string, owner *models.User) (*models.FileMeta, error)
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
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := fms.db.CreateFileMeta(dbCtx, fm); err != nil {
		return fmt.Errorf("[services.SaveFileMeta] failed to save file metadata: %w", err)
	}
	return nil
}

func (fms *fileMetaService) LookupFileMeta(ctx context.Context, id string, owner *models.User) (*models.FileMeta, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	fm, err := fms.db.GetFileMeta(dbCtx, id, owner)
	if err != nil {
		return nil, fmt.Errorf("[services.LookupFileMeta] failed to get file for id '%s': %w", id, err)
	}
	return fm, nil
}

func (fms *fileMetaService) LookupFileMetaByNameAndOwner(ctx context.Context, name string, owner *models.User) (*models.FileMeta, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	fm, err := fms.db.GetFileMetaByNameAndOwner(dbCtx, name, owner)
	if err != nil {
		return nil, fmt.Errorf("[services.LookupFileMeta] failed to get file '%s' for user '%s': %w", name, owner, err)
	}
	return fm, nil
}

func (fms *fileMetaService) LookupAllFileMeta(ctx context.Context, owner *models.User) ([]*models.FileMeta, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	data, err := fms.db.GetAllFileMeta(dbCtx, owner)
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
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := fms.db.DeleteFileMeta(dbCtx, id); err != nil {
		return fmt.Errorf("[services.DeleteFileMeta] failed to delete file metadata: %w", err)
	}
	return nil
}
