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
	"github.com/google/uuid"
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
	return &FileService{db: fileRepo, storage: storageRepo, thumbnailer: thumbnailer, TmpStorage: tmpStorage}
}

func (fs *FileService) ProcessBatch(ctx context.Context, batch []*models.FileMeta) []error {
	ch := make(chan error)
	wg := sync.WaitGroup{}
	proccessingErrors := []error{}

	for _, item := range batch {
		wg.Add(1)
		go func(fm *models.FileMeta) {
			defer wg.Done()

			thumbReader, err := fs.thumbnailer.Generate(ctx, fm)
			if err != nil {
				ch <- fmt.Errorf("services.ProcessBatch: failed to generate thumbnail for %s: %w", fm.Name, err)
				return
			}

			if thumbReader != nil {
				tfm := &models.FileMeta{
					ID:          fmt.Sprintf("thumb-%s", fm.ID),
					ParentID:    fm.ID,
					Name:        fmt.Sprintf("thumb-%s", fm.Name),
					ContentType: "image/jpeg",
					Owner:       fm.Owner,
				}

				var path string
				path, err = utils.StageFileToDisk(ctx, fs.TmpStorage, tfm.ID, thumbReader)
				if err != nil {
					ch <- fmt.Errorf("services.Processbatch: failed to stage thumbnail for %s to disk: %w", fm.Name, err)
					return
				}
				tfm.TmpFilePath = path

				if err = fs.storage.Upload(ctx, tfm.ID, tfm.TmpFilePath); err != nil {
					ch <- fmt.Errorf("upload failed for %s: %w", fm.Name, err)
					return
				}

				if err = fs.SaveFileMeta(ctx, tfm); err != nil {
					if rbErr := fs.DeleteFile(ctx, tfm.ID); rbErr != nil {
						// TODO replace w proper logging
						fmt.Printf("CRITICAL: failed to delete orphaned file %s from storage: %v\n", fm.Name, rbErr)
					}
					ch <- fmt.Errorf("save metadata failed for %s: %w", fm.Name, err)
					return
				}

				err = os.Remove(tfm.TmpFilePath)
				if err != nil {
					fmt.Printf("failed to remove tmp %s: %v", tfm.Name, err)
				}
			}

			if err = fs.storage.Upload(ctx, fm.ID, fm.TmpFilePath); err != nil {
				ch <- fmt.Errorf("upload failed for %s: %w", fm.Name, err)
				return
			}

			if err = fs.SaveFileMeta(ctx, fm); err != nil {
				if rbErr := fs.DeleteFile(ctx, fm.ID); rbErr != nil {
					// TODO replace w proper logging
					fmt.Printf("CRITICAL: failed to delete orphaned file %s from storage: %v\n", fm.Name, rbErr)
				}
				ch <- fmt.Errorf("save metadata failed for %s: %w", fm.Name, err)
				return
			}

			err = os.Remove(fm.TmpFilePath)
			if err != nil {
				fmt.Printf("failed to remove tmpfile: %v", err)
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
	fm, cerr := fs.LookupFileMeta(ctx, id)
	if cerr != nil {
		cerr = errors.Join(fmt.Errorf("services.DeleteFile: failed to find file for %s: %w", id, cerr))
	}

	if err := fs.storage.Delete(ctx, fm.ID); err != nil {
		cerr = errors.Join(fmt.Errorf("services.DeleteFile: failed to delete file %s from storage: %w", id, err))
	}

	if err := fs.DeleteFileMeta(ctx, fm.ID); err != nil {
		cerr = errors.Join(fmt.Errorf("services.DeleteFile: failed to delete file metadata for %s: %w", id, err))
	}

	if fm.ParentID != "" {
		if err := fs.storage.Delete(ctx, fm.ParentID); err != nil {
			cerr = errors.Join(fmt.Errorf("services.DeleteFile: failed to delete file %s from storage: %w", id, err))
		}

		if err := fs.DeleteFileMeta(ctx, fm.ParentID); err != nil {
			cerr = errors.Join(fmt.Errorf("services.DeleteFile: failed to delete file metadata for %s: %w", id, err))
		}
	}

	if cerr != nil {
		return cerr
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

func (fs *FileService) DeleteFileMeta(ctx context.Context, id string) error {
	if err := fs.db.Delete(ctx, id); err != nil {
		return fmt.Errorf("services.DeleteFileMeta: failed to delete file metadata: %w", err)
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
