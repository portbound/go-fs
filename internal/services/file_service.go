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
	ProcessBatch(ctx context.Context, batch []*models.FileMeta) []error
	DownloadFile(ctx context.Context, id string) (io.ReadCloser, error)
	DeleteFile(ctx context.Context, id string) error
	StageFileToDisk(ctx context.Context, fileName string, reader io.Reader) (string, int64, error)
}

type fileService struct {
	storage         repositories.StorageRepository
	fileMetaService FileMetaService
	tmpDir          string
}

func NewFileService(storageRepo repositories.StorageRepository, fileMetaService FileMetaService, tmpDir string) FileService {
	return &fileService{
		storage:         storageRepo,
		fileMetaService: fileMetaService,
		tmpDir:          tmpDir,
	}
}

func (fs *fileService) ProcessBatch(ctx context.Context, batch []*models.FileMeta) []error {
	var wg sync.WaitGroup
	var batchErrs []error

	ch := make(chan error)
	for _, fm := range batch {
		wg.Add(1)
		go func() {
			defer wg.Done()

			existing, err := fs.fileMetaService.LookupFileMeta(ctx, fm.ID)
			if err != nil {
				if !errors.Is(err, sql.ErrNoRows) {
					ch <- fmt.Errorf("[services.ProcessBatch] '%s': %w", fm.Name, err)
					return
				}
			}
			if existing != nil {
				ch <- fmt.Errorf("[services.ProcessBatch] file %s already exists. Skipping.", fm.Name)
				return
			}

			thumbnailReader, err := GenerateThumbnail(ctx, fm)
			if err != nil {
				ch <- fmt.Errorf("[services.ProcessBatch] failed to generate thumbnail for '%s': %w", fm.Name, err)
				return
			}

			thumbID := fmt.Sprintf("thumb-%s", fm.ID)
			path, _, err := fs.StageFileToDisk(ctx, thumbID, thumbnailReader)
			if err != nil {
				ch <- fmt.Errorf("[services.ProcessBatch] failed to stage thumbnail for %s to disk: %w", fm.Name, err)
			}
			defer os.Remove(path)

			fm.ThumbID = thumbID
			thumbFm := &models.FileMeta{
				ID:          thumbID,
				ParentID:    fm.ID,
				ThumbID:     "",
				Name:        fmt.Sprintf("thumb-%s", fm.Name),
				ContentType: "image/jpeg",
				// Size:        bytesWritten,
				// UploadDate:  time.Now(),
				Owner:       fm.Owner,
				TmpFilePath: path,
			}

			if err = fs.processFile(ctx, thumbFm); err != nil {
				ch <- fmt.Errorf("[services.ProcessBatch] failed to process thumbnail for %s: %w", fm.Name, err)
				return
			}

			// TODO setup logger
			// msg := fmt.Sprintf("Thumbnail Upload Success: File '%s'", fm.Name)
			// fs.logger.Write(msg)

			fileReader, err := os.Open(fm.TmpFilePath)
			if err != nil {
				ch <- fmt.Errorf("[services.ProcessBatch] failed to open %s: %w", fm.TmpFilePath, err)
				return
			}
			defer fileReader.Close()

			if err := fs.processFile(ctx, fm); err != nil {
				if fm.ThumbID != "" {
					if err = fs.DeleteFile(ctx, fm.ThumbID); err != nil {
						// TODO setup logger
						// msg := fmt.Sprintf("CRITICAL - Delete File: Failed to delete orphaned thumbnail '%s'", fm.ThumbID)
						// fs.logger.Write(msg)
						ch <- fmt.Errorf("CRITICAL - [services.ProcessBatch] failed to delete orphaned thumbnail %s: %v", fm.ThumbID, err)
					}
				}
				ch <- fmt.Errorf("[services.ProcessBatch] failed to process %s: %w", fm.Name, err)
				return
			}

			// TODO setup logger
			// msg = fmt.Sprintf("File Upload Success: File '%s'", fm.Name)
			// fs.logger.Write(msg)
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

func (fs *fileService) DownloadFile(ctx context.Context, id string) (io.ReadCloser, error) {
	gcsReader, err := fs.storage.Download(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("[services.GetFile] failed to get file from storage: %w", err)
	}

	return gcsReader, nil
}

func (fs *fileService) DeleteFile(ctx context.Context, id string) error {
	if err := fs.storage.Delete(ctx, id); err != nil {
		return fmt.Errorf("services.DeleteFile: failed to delete %s from storage: %w", id, err)
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

func (fs *fileService) processFile(ctx context.Context, fm *models.FileMeta) error {
	var err error
	fm.Size, fm.UploadDate, err = fs.storage.Upload(ctx, fm.ID, fm.TmpFilePath)
	if err != nil {
		return fmt.Errorf("[fileService.processFile] upload failed for %s: %w", fm.Name, err)
	}

	if err := fs.fileMetaService.SaveFileMeta(ctx, fm); err != nil {
		if rbErr := fs.DeleteFile(ctx, fm.ID); rbErr != nil {
			// TODO setup logger
			// msg := fmt.Sprintf("CRITICAL: failed to delete orphaned file %s from storage: %v", fm.Name, rbErr)
			// fs.logger.Write(msg)
		}
		return fmt.Errorf("[fileService.processFile] save metadata failed for %s: %w", fm.Name, err)
	}

	return nil
}
