// Package services
package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"cloud.google.com/go/storage"
	"github.com/portbound/go-fs/internal/models"
	"github.com/portbound/go-fs/internal/repositories"
	"github.com/portbound/go-fs/internal/utils"
)

type FileService struct {
	db          repositories.FileRepository
	storage     repositories.StorageRepository
	thumbnailer *ThumbnailService
	TmpStorage  string
}

func NewFileService(fileRepo repositories.FileRepository, storageRepo repositories.StorageRepository, thumbnailer *ThumbnailService, tmpStorage string) *FileService {
	return &FileService{
		db:          fileRepo,
		storage:     storageRepo,
		thumbnailer: thumbnailer,
		TmpStorage:  tmpStorage,
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

			thumbnailReader, err := fs.thumbnailer.Generate(ctx, fm)
			if err != nil {
				ch <- fmt.Errorf("services.ProcessBatch: failed to generate thumbnail for '%s': %w", fm.Name, err)
				return
			}

			if thumbnailReader != nil {
				tfm := &models.FileMeta{
					ID:          fmt.Sprintf("thumb-%s", fm.ID),
					ParentID:    fm.ID,
					ThumbID:     "",
					Name:        fmt.Sprintf("thumb-%s", fm.Name),
					ContentType: "image/jpeg",
					Owner:       fm.Owner,
				}
				fm.ThumbID = tfm.ID

				tfm.TmpFilePath, err = utils.StageFileToDisk(ctx, fs.TmpStorage, tfm.ID, thumbnailReader)
				if err != nil {
					ch <- fmt.Errorf("services.Processbatch: failed to stage thumbnail for %s to disk: %w", fm.Name, err)
				}
				defer os.Remove(tfm.TmpFilePath)

				if err = fs.processFile(ctx, tfm, thumbnailReader); err != nil {
					ch <- fmt.Errorf("services.ProcessBatch: failed to process thumbnail for %s: %w", fm.Name, err)
					return
				}
			}

			fileReader, err := os.Open(fm.TmpFilePath)
			if err != nil {
				ch <- fmt.Errorf("services.ProcessBatch: failed to open %s: %w", fm.TmpFilePath, err)
				return
			}
			defer fileReader.Close()

			if err := fs.processFile(ctx, fm, fileReader); err != nil {
				if fm.ThumbID != "" {
					if err = fs.DeleteFile(ctx, fm.ThumbID); err != nil {
						e := fmt.Sprintf("CRITICAL services.ProcessBatch: failed to delete orphaned thumbnail %s: %v", fm.ThumbID, err)
						// TODO replace w proper logging
						fmt.Println(e)
						ch <- errors.New(e)
					}
				}
				ch <- fmt.Errorf("services:ProcessBatch: failed to process %s: %w", fm.Name, err)
				return
			}

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

func (fs *FileService) GetThumbnails(ctx context.Context) ([]string, error) {
	fileNames, err := fs.storage.ListObjects(ctx, &storage.Query{Prefix: "thumb-"})
	if err != nil {
		return nil, fmt.Errorf("services.GetBatch: failed to get fileNames from storage: %w", err)
	}

	return fileNames, nil
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

func (fs *FileService) saveFileMeta(ctx context.Context, fm *models.FileMeta) error {
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

func (fs *FileService) processFile(ctx context.Context, fm *models.FileMeta, fileReader io.Reader) error {
	// if fileReader != nil {
	// 	path, err := utils.StageFileToDisk(ctx, fs.TmpStorage, fm.ID, fileReader)
	// 	if err != nil {
	// 		return fmt.Errorf("services.Processbatch: failed to stage thumbnail for %s to disk: %w", fm.Name, err)
	// 	}
	// 	fm.TmpFilePath = path
	// 	defer os.Remove(fm.TmpFilePath)
	//
	if err := fs.storage.Upload(ctx, fm.ID, fm.TmpFilePath); err != nil {
		return fmt.Errorf("upload failed for %s: %w", fm.Name, err)
	}

	if err := fs.saveFileMeta(ctx, fm); err != nil {
		if rbErr := fs.DeleteFile(ctx, fm.ID); rbErr != nil {
			// TODO replace w proper logging
			fmt.Printf("CRITICAL: failed to delete orphaned file %s from storage: %v", fm.Name, rbErr)
			return fmt.Errorf("CRITICAL: failed to delete orphaned file %s from storage: %w", fm.Name, rbErr)
		}
		return fmt.Errorf("save metadata failed for %s: %w", fm.Name, err)
	}
	// }

	return nil
}
