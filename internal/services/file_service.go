// Package services
package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/portbound/go-fs/internal/models"
	"github.com/portbound/go-fs/internal/repositories"
	"github.com/portbound/go-fs/internal/utils"
)

type FileService struct {
	db         repositories.FileRepository
	storage    repositories.StorageRepository
	TmpStorage string
}

func NewFileService(fileRepo repositories.FileRepository, storageRepo repositories.StorageRepository, tmpStorage string) *FileService {
	return &FileService{db: fileRepo, storage: storageRepo, TmpStorage: tmpStorage}
}

func (fs *FileService) GetFile(ctx context.Context, id uuid.UUID) (*models.FileMeta, io.ReadCloser, error) {
	fm, err := fs.LookupFileMeta(ctx, id)
	if err != nil {
		return nil, nil, fmt.Errorf("services.GetFile: failed to lookup file metadata: %w", err)
	}

	gcsReader, err := fs.storage.Download(ctx, fm)
	if err != nil {
		return nil, nil, fmt.Errorf("services.GetFile: failed to get file from storage: %w", err)
	}

	return fm, gcsReader, nil
}

func (fs *FileService) ProcessBatch(ctx context.Context, batch []*models.FileMeta) []error {
	ch := make(chan error)
	wg := sync.WaitGroup{}
	proccessingErrors := []error{}

	for _, item := range batch {
		wg.Add(1)
		go func(fm *models.FileMeta) {
			defer wg.Done()
			fileType := strings.ToLower(strings.Split(fm.ContentType, "/")[0])
			fileSubType := strings.ToLower(strings.Split(fm.ContentType, "/")[1])

			if fileType == "image" {
				switch fileSubType {
				case "jpg", "png", "gif":
					r, err := CreateThumbnail(ctx, fm.TmpFilePath)
					if err != nil {
						ch <- fmt.Errorf("services.UploadBatch: failed to create thumbnail for %s: %w", fm.Name, err)
						return
					}

					fileName := fmt.Sprintf("thumbnail-%s.jpg", fm.Name)
					path, err := utils.StageFileToDisk(ctx, fs.TmpStorage, fileName, r)
					if err != nil {
						ch <- fmt.Errorf("services.Processbatch: failed to stage thumbnail for %s to disk: %w", fm.Name, err)
						return
					}
					fm.TmpThumbPath = path
				default:
					ch <- errors.New("services.ProcessBatch: failed to create thumnail for %s: file type not supported")
					return
				}
			}

			if err := fs.storage.Upload(ctx, fm.TmpFilePath); err != nil {
				ch <- fmt.Errorf("upload failed for %s: %w", fm.Name, err)
				return
			}

			if err := fs.storage.Upload(ctx, fm.TmpThumbPath); err != nil {
				if rbErr := fs.DeleteFile(ctx, fm); rbErr != nil {
					// TODO replace w proper logging
					fmt.Printf("CRITICAL: failed to delete orphaned file %s from storage: %v\n", fm.Name, rbErr)
				}
				ch <- fmt.Errorf("services.Processbatch: failed to upload thumbnail for %s: %w", fm.Name, err)
				return
			}

			if err := fs.SaveFileMeta(ctx, fm); err != nil {
				if rbErr := fs.DeleteFile(ctx, fm); rbErr != nil {
					// TODO replace w proper logging
					fmt.Printf("CRITICAL: failed to delete orphaned file %s from storage: %v\n", fm.Name, rbErr)
				}
				if rbErr := fs.DeleteFile(ctx, fm); rbErr != nil {
					// TODO replace w proper logging
					fmt.Printf("CRITICAL: failed to delete orphaned file %s from storage: %v\n", fm.Name, rbErr)
				}
				ch <- fmt.Errorf("save metadata failed for %s: %w", fm.Name, err)
				return
			}

			err := os.Remove(fm.TmpFilePath)
			if err != nil {
				fmt.Printf("failed to remove tmpfile: %v", err)
			}
			err = os.Remove(fm.TmpThumbPath)
			if err != nil {
				fmt.Printf("failed to remove tmpthumb: %v", err)
			}

			ch <- nil
		}(item)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	for err := range ch {
		if err != nil {
			proccessingErrors = append(proccessingErrors, err)
		}
	}

	return proccessingErrors
}

func (fs *FileService) DeleteFile(ctx context.Context, fm *models.FileMeta) error {
	if err := fs.storage.Delete(ctx, fm.Name); err != nil {
		return fmt.Errorf("services.DeleteFile: failed to delete file from storage: %w", err)
	}

	if err := fs.DeleteFileMeta(ctx, fm.ID); err != nil {
		// TODO replace w proper logging
		return fmt.Errorf("CRITICAL: services.DeleteFileMeta: failed to delete orphaned file metadata: %w", err)
	}
	return nil
}

func (fs *FileService) DeleteBatch(ctx context.Context, ids *[]uuid.UUID) []error {

	return nil
}

func (fs *FileService) SaveFileMeta(ctx context.Context, fm *models.FileMeta) error {
	if err := fs.db.Create(ctx, fm); err != nil {
		return fmt.Errorf("services.SaveFileMeta: failed to save file metadata: %w", err)
	}
	return nil
}

func (fs *FileService) DeleteFileMeta(ctx context.Context, id uuid.UUID) error {
	if err := fs.db.Delete(ctx, id); err != nil {
		return fmt.Errorf("services.DeleteFileMeta: failed to delete file metadata: %w", err)
	}
	return nil
}

func (fs *FileService) LookupFileMeta(ctx context.Context, id uuid.UUID) (*models.FileMeta, error) {
	fm, err := fs.db.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("services.LookupFileMeta: failed to get file for id '%s': %w", id, err)
	}
	return fm, nil
}
