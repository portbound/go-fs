// Package services
package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/portbound/go-fs/internal/models"
	"github.com/portbound/go-fs/internal/repositories"
	"github.com/portbound/go-fs/internal/utils"
)

type FileService struct {
	db               repositories.FileRepository
	storage          repositories.StorageRepository
	thumbnailService *ThumbnailService
	TmpDir           string
}

func NewFileService(fileRepo repositories.FileRepository, storageRepo repositories.StorageRepository, tmpDir string) *FileService {
	return &FileService{
		db:               fileRepo,
		storage:          storageRepo,
		thumbnailService: NewThumbnailService(),
		TmpDir:           tmpDir,
	}
}

func (fs *FileService) ProcessBatch(ctx context.Context, batch []*models.FileMeta) []error {
	var wg sync.WaitGroup
	var batchErrs []error

	ch := make(chan error)
	for _, fm := range batch {
		wg.Add(1)
		go func() {
			defer wg.Done()

			existing, err := fs.LookupFileMeta(ctx, fm.ID)
			if err != nil {
				if !errors.Is(err, sql.ErrNoRows) {
					ch <- fmt.Errorf("services.ProcessBatch: '%s': %w", fm.Name, err)
					return
				}
			}
			if existing != nil {
				ch <- fmt.Errorf("services.ProcessBatch: file %s already exists. Skipping.", fm.Name)
				return
			}

			thumbnailReader, err := fs.thumbnailService.Generate(ctx, fm)
			if err != nil {
				ch <- fmt.Errorf("services.ProcessBatch: failed to generate thumbnail for '%s': %w", fm.Name, err)
				return
			}

			thumbID := fmt.Sprintf("thumb-%s", fm.ID)
			path, bytesWritten, err := utils.StageFileToDisk(ctx, fs.TmpDir, thumbID, thumbnailReader)
			if err != nil {
				ch <- fmt.Errorf("services.Processbatch: failed to stage thumbnail for %s to disk: %w", fm.Name, err)
			}
			defer os.Remove(path)

			fm.ThumbID = thumbID
			thumbFm := &models.FileMeta{
				ID:          thumbID,
				ParentID:    fm.ID,
				ThumbID:     "",
				Name:        fmt.Sprintf("thumb-%s", fm.Name),
				ContentType: "image/jpeg",
				Size:        bytesWritten,
				UploadDate:  time.Now(),
				Owner:       fm.Owner,
				TmpFilePath: path,
			}

			if err = fs.processFile(ctx, thumbFm); err != nil {
				ch <- fmt.Errorf("services.ProcessBatch: failed to process thumbnail for %s: %w", fm.Name, err)
				return
			}

			// fs.logger.Writer.Printf("Thumbnail Upload Success: File '%s'", fm.Name)

			fileReader, err := os.Open(fm.TmpFilePath)
			if err != nil {
				ch <- fmt.Errorf("services.ProcessBatch: failed to open %s: %w", fm.TmpFilePath, err)
				return
			}
			defer fileReader.Close()

			if err := fs.processFile(ctx, fm); err != nil {
				if fm.ThumbID != "" {
					if err = fs.DeleteFile(ctx, fm.ThumbID); err != nil {
						// fs.logger.Writer.Printf("CRITICAL - Delete File: Failed to delete orphaned thumbnail '%s'", fm.ThumbID)
						ch <- fmt.Errorf("CRITICAL - services.ProcessBatch: failed to delete orphaned thumbnail %s: %v", fm.ThumbID, err)
					}
				}
				ch <- fmt.Errorf("services:ProcessBatch: failed to process %s: %w", fm.Name, err)
				return
			}

			// fs.logger.Writer.Printf("File Upload Success: file '%s'", fm.Name)
			ch <- nil
		}()
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	for err := range ch {
		if err != nil {
			batchErrs = append(batchErrs, err)
		}
	}

	return batchErrs
}

func (fs *FileService) GetFile(ctx context.Context, id string) (io.ReadCloser, error) {
	gcsReader, err := fs.storage.Download(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("services.GetFile: failed to get file from storage: %w", err)
	}

	return gcsReader, nil
}

func (fs *FileService) DeleteFile(ctx context.Context, id string) error {
	if err := fs.storage.Delete(ctx, id); err != nil {
		return fmt.Errorf("services.DeleteFile: failed to delete %s from storage: %w", id, err)
	}
	return nil
}

func (fs *FileService) LookupFileMeta(ctx context.Context, id string) (*models.FileMeta, error) {
	fm, err := fs.db.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("services.LookupFileMeta: failed to get file for id '%s': %w", id, err)
	}
	return fm, nil
}

func (fs *FileService) LookupAllFileMeta(ctx context.Context) ([]*models.FileMeta, error) {
	data, err := fs.db.GetAll(ctx)
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

func (fs *FileService) SaveFileMeta(ctx context.Context, fm *models.FileMeta) error {
	if err := fs.db.Create(ctx, fm); err != nil {
		return fmt.Errorf("services.SaveFileMeta: failed to save file metadata: %w", err)
	}
	return nil
}

func (fs *FileService) DeleteFileMeta(ctx context.Context, id string) error {
	if err := fs.db.Delete(ctx, id); err != nil {
		return fmt.Errorf("services.DeleteFileMeta: failed to delete file metadata: %w", err)
	}
	return nil
}

func (fs *FileService) processFile(ctx context.Context, fm *models.FileMeta) error {
	var err error
	fm.Size, fm.UploadDate, err = fs.storage.Upload(ctx, fm.ID, fm.TmpFilePath)
	if err != nil {
		return fmt.Errorf("upload failed for %s: %w", fm.Name, err)
	}

	if err := fs.SaveFileMeta(ctx, fm); err != nil {
		if rbErr := fs.DeleteFile(ctx, fm.ID); rbErr != nil {
			return fmt.Errorf("CRITICAL: failed to delete orphaned file %s from storage: %w", fm.Name, rbErr)
		}
		return fmt.Errorf("save metadata failed for %s: %w", fm.Name, err)
	}

	return nil
}
