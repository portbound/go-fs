// Package services
package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/portbound/go-fs/internal/models"
	"github.com/portbound/go-fs/internal/repositories"
)

type FileService struct {
	db               repositories.FileRepository
	storage          repositories.StorageRepository
	localStoragePath string
}

func NewFileService(fileRepo repositories.FileRepository, storageRepo repositories.StorageRepository, localStoragePath string) *FileService {
	return &FileService{db: fileRepo, storage: storageRepo, localStoragePath: localStoragePath}
}

func (fs *FileService) GetFile(ctx context.Context, id uuid.UUID) (*models.FileMeta, io.ReadCloser, error) {
	fm, err := fs.lookupFileMeta(ctx, id)
	if err != nil {
		return nil, nil, fmt.Errorf("services.GetFile: failed to lookup file metadata: %w", err)
	}

	gcsReader, err := fs.storage.Download(ctx, fm)
	if err != nil {
		return nil, nil, fmt.Errorf("services.GetFile: failed to get file from storage: %w", err)
	}

	return fm, gcsReader, nil
}

func (fs *FileService) UploadBatch(ctx context.Context, batch []*models.FileMeta) []error {
	type result struct {
		fm  *models.FileMeta
		err error
	}

	ch := make(chan *result)
	wg := sync.WaitGroup{}
	proccessingErrors := []error{}

	for _, item := range batch {
		wg.Add(1)
		go func(fm *models.FileMeta) {
			defer wg.Done()
			mime := strings.ToLower(strings.TrimSuffix(item.ContentType, "/"))

			if mime == "image" {

			}
			if mime == "video" {
				// create thumbnail
				// create preview
			}

			if err := fs.storage.Upload(ctx, fm); err != nil {
				ch <- &result{fm: fm, err: fmt.Errorf("upload failed for %s: %w", fm.Name, err)}
				return
			}

			if err := fs.saveFilemeta(ctx, fm); err != nil {
				if delErr := fs.DeleteFile(ctx, fm.ID); delErr != nil {
					fmt.Printf("CRITICAL: failed to delete orphaned file %s from storage: %v\n", fm.Name, delErr)
				}
				ch <- &result{fm: fm, err: fmt.Errorf("save metadata failed for %s: %w", fm.Name, err)}
				return
			}

			ch <- &result{fm: fm, err: nil}
			os.Remove(fm.TmpDir)
		}(item)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	for res := range ch {
		if res.err != nil {
			proccessingErrors = append(proccessingErrors, res.err)
		}
	}

	return proccessingErrors
}

func (fs *FileService) DeleteFile(ctx context.Context, id uuid.UUID) error {
	fm, err := fs.lookupFileMeta(ctx, id)
	if err != nil {
		return fmt.Errorf("services.DeleteFile: failed to lookup file metadata: %w", err)
	}

	if err := fs.storage.Delete(ctx, fm); err != nil {
		return fmt.Errorf("services.DeleteFile: failed to delete file from storage: %w", err)
	}

	if err := fs.deleteFileMeta(ctx, id); err != nil {
		return fmt.Errorf("services.DeleteFile: successfully deleted file, but failed to delete file metadata for id %s: %w", id, err)
	}

	return nil
}

func (fs *FileService) DeleteBatch(ctx context.Context, ids *[]uuid.UUID) []error {

	return nil
}

func (fs *FileService) StageFileToDisk(ctx context.Context, metadata *models.FileMeta, reader io.Reader) error {
	type result struct {
		bytes int64
		err   error
	}

	tmpDir := filepath.Join(fs.localStoragePath, "tmp")

	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("services.NewFileService: failed to create tmp storage dir: %w", err)
	}

	metadata.ID = uuid.New()
	tmpFileName := fmt.Sprintf("%s-%s", metadata.ID.String(), strings.ReplaceAll(metadata.Name, " ", "_"))
	tmpFilePath := filepath.Join(tmpDir, tmpFileName)

	tmpFile, err := os.Create(tmpFilePath)
	if err != nil {
		return fmt.Errorf("services.StageFileToDisk: failed to create tmp file: %w", err)
	}
	defer tmpFile.Close()

	ch := make(chan result, 1)
	go func() {
		bytes, err := io.Copy(tmpFile, reader)
		ch <- result{bytes: bytes, err: err}
	}()

	select {
	case <-ctx.Done():
		os.Remove(tmpFilePath)
		return ctx.Err()
	case result := <-ch:
		if result.err != nil {
			os.Remove(tmpFilePath)
			return fmt.Errorf("services.StageFileToDisk: failed to write to tmp file: %w", result.err)
		}
		metadata.Size = result.bytes
		metadata.TmpDir = tmpFilePath
	}

	return nil
}

func (fs *FileService) saveFilemeta(ctx context.Context, fm *models.FileMeta) error {
	if err := fs.db.Create(ctx, fm); err != nil {
		return fmt.Errorf("services.SaveFileMeta: failed to save file metadata: %w", err)
	}
	return nil
}

func (fs *FileService) deleteFileMeta(ctx context.Context, id uuid.UUID) error {
	if err := fs.db.Delete(ctx, id); err != nil {
		return fmt.Errorf("services.DeleteFileMeta: failed to delete file metadata: %w", err)
	}
	return nil
}

func (fs *FileService) lookupFileMeta(ctx context.Context, id uuid.UUID) (*models.FileMeta, error) {
	fm, err := fs.db.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("services.LookupFileMeta: failed to get file for id '%s': %w", id, err)
	}
	return fm, nil
}
