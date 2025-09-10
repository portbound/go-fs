// Package service
package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/portbound/go-fs/internal/models"
	"github.com/portbound/go-fs/internal/repositories"
)

type FileService interface {
	ProcessBatch(ctx context.Context, batch []*models.FileMeta, user *models.User) []error
	DownloadFile(ctx context.Context, id string, owner *models.User) (io.ReadCloser, error)
	DeleteFile(ctx context.Context, id string, owner *models.User) error
	StageFileToDisk(ctx context.Context, fileName string, reader io.Reader) (string, int64, error)
}

type fileService struct {
	storage repositories.StorageRepository
	fms     FileMetaService
	tmpDir  string
}

func NewFileService(storageRepo repositories.StorageRepository, fileMetaService FileMetaService, tmpDir string) FileService {
	return &fileService{
		storage: storageRepo,
		fms:     fileMetaService,
		tmpDir:  tmpDir,
	}
}

func (fs *fileService) ProcessBatch(ctx context.Context, batch []*models.FileMeta, owner *models.User) []error {
	var wg sync.WaitGroup
	var batchErrs []error

	ch := make(chan error)
	for _, fm := range batch {
		wg.Go(func() {
			// TODO wrap this in a function so we can check for filename + bucket uniqueness
			_, err := fs.fms.LookupFileMeta(ctx, fm.ID, owner)
			if err == nil {
				ch <- fmt.Errorf("[services.ProcessBatch] file %s already exists (skipping)", fm.Name)
				return
			}
			if !errors.Is(err, sql.ErrNoRows) {
				ch <- fmt.Errorf("[services.ProcessBatch] '%s': %w", fm.Name, err)
				return
			}

			thumbReader, err := GenerateThumbnail(ctx, fm)
			if err != nil {
				ch <- fmt.Errorf("[services.ProcessBatch] failed to generate thumbnail for '%s': %w", fm.Name, err)
				return
			}

			thumbID := fmt.Sprintf("thumb-%s", fm.ID)
			tfm := &models.FileMeta{
				ID:          thumbID,
				ParentID:    fm.ID,
				ThumbID:     "",
				Name:        fmt.Sprintf("thumb-%s", fm.Name),
				ContentType: "image/jpeg",
				Owner:       fm.Owner,
			}

			if err = fs.processFile(ctx, tfm, owner, thumbReader); err != nil {
				ch <- fmt.Errorf("[services.ProcessBatch] failed to process thumbnail for %s: %w", fm.Name, err)
				return
			}

			fm.ThumbID = thumbID
			fileReader, err := os.Open(fm.TmpFilePath)
			if err != nil {
				ch <- fmt.Errorf("[services.ProcessBatch] failed to open %s: %w", fm.TmpFilePath, err)
				return
			}
			defer fileReader.Close()

			if err := fs.processFile(ctx, fm, owner, fileReader); err != nil {
				if fm.ThumbID != "" {
					if err = fs.DeleteFile(ctx, fm.ThumbID, owner); err != nil {
						// TODO setup logger
						ch <- fmt.Errorf("CRITICAL - [services.ProcessBatch] failed to delete orphaned thumbnail %s: %v", fm.ThumbID, err)
					}
				}
				ch <- fmt.Errorf("[services.ProcessBatch] failed to process %s: %w", fm.Name, err)
				return
			}

			// TODO setup logger
			ch <- nil
		})
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

func (fs *fileService) DownloadFile(ctx context.Context, id string, owner *models.User) (io.ReadCloser, error) {
	gcsReader, err := fs.storage.Download(ctx, id, owner.BucketName)
	if err != nil {
		return nil, fmt.Errorf("[services.GetFile] failed to get file from storage: %w", err)
	}

	return gcsReader, nil
}

func (fs *fileService) DeleteFile(ctx context.Context, id string, owner *models.User) error {
	if err := fs.storage.Delete(ctx, id, owner.BucketName); err != nil {
		return fmt.Errorf("[services.DeleteFile] failed to delete %s from storage: %w", id, err)
	}
	return nil
}

func (fs *fileService) StageFileToDisk(ctx context.Context, fileName string, reader io.Reader) (string, int64, error) {
	type chanl struct {
		bytesWritten int64
		err          error
	}

	path := fs.tmpDir
	if err := os.MkdirAll(path, 0755); err != nil {
		return "", 0, fmt.Errorf("[util.StageFileToDisk] failed to create storage dir at '%s': %w", path, err)
	}

	file, err := os.Create(filepath.Join(path, fileName))
	if err != nil {
		return "", 0, fmt.Errorf("[util.StageFileToDisk] failed to create temp file: %w", err)
	}
	defer file.Close()

	ch := make(chan *chanl, 1)
	go func() {
		bytesWritten, copyErr := io.Copy(file, reader)
		ch <- &chanl{bytesWritten: bytesWritten, err: copyErr}
	}()

	select {
	case <-ctx.Done():
		os.Remove(file.Name())
		return "", 0, ctx.Err()
	case result := <-ch:
		if result.err != nil {
			os.Remove(file.Name())
			return "", 0, fmt.Errorf("[fileService.StageFileToDisk] failed to write to tmp file: %w", result.err)
		}
		return file.Name(), result.bytesWritten, nil
	}
}

func (fs *fileService) processFile(ctx context.Context, fm *models.FileMeta, owner *models.User, src io.Reader) error {
	var err error
	fm.Size, fm.UploadDate, err = fs.storage.Upload(ctx, fm.ID, owner.BucketName, src)
	if err != nil {
		return fmt.Errorf("[fileService.processFile] upload failed for %s: %w", fm.Name, err)
	}

	if err := fs.fms.SaveFileMeta(ctx, fm); err != nil {
		if rbErr := fs.DeleteFile(ctx, fm.ID, owner); rbErr != nil {
			// TODO setup logger
		}
		return fmt.Errorf("[fileService.processFile] save metadata failed for %s: %w", fm.Name, err)
	}

	// TODO setup logger

	return nil
}
