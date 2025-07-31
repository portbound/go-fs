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
	fileRepo         repositories.FileRepository
	storageRepo      repositories.StorageRepository
	localStoragePath string
}

func NewFileService(fileRepo repositories.FileRepository, storageRepo repositories.StorageRepository, localStoragePath string) *FileService {
	return &FileService{fileRepo: fileRepo, storageRepo: storageRepo, localStoragePath: localStoragePath}
}

func (fs *FileService) UploadFile(ctx context.Context, fm *models.FileMeta) error {
	if err := fs.storageRepo.Upload(ctx, fm); err != nil {
		return fmt.Errorf("services.UploadFile: failed to upload file to storage: %w", err)
	}

	os.Remove(fm.TmpDir)
	return nil
}

func (fs *FileService) DeleteFile(ctx context.Context, fm *models.FileMeta) error {
	if err := fs.storageRepo.Delete(ctx, fm); err != nil {
		return fmt.Errorf("services.DeleteFile: failed to delete file from storage: %w", err)
	}
	return nil
}

func (fs *FileService) ProcessBatch(ctx context.Context, batch []*models.FileMeta) []error {
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
			if err := fs.UploadFile(ctx, fm); err != nil {
				ch <- &result{fm: fm, err: fmt.Errorf("upload failed for %s: %w", fm.Name, err)}
				return
			}

			if err := fs.SaveFileMeta(ctx, fm); err != nil {
				if delErr := fs.DeleteFile(ctx, fm); delErr != nil {
					fmt.Printf("CRITICAL: failed to delete orphaned file %s from storage: %v\n", fm.Name, delErr)
				}
				ch <- &result{fm: fm, err: fmt.Errorf("save metadata failed for %s: %w", fm.Name, err)}
				return
			}

			ch <- &result{fm: fm, err: nil}
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

func (fs *FileService) SaveFileMeta(ctx context.Context, fm *models.FileMeta) error {
	if err := fs.fileRepo.Create(ctx, fm); err != nil {
		return fmt.Errorf("services.SaveFileMeta: failed to save file metadata: %w", err)
	}
	return nil
}

func (fs *FileService) DeleteFileMeta(ctx context.Context, id uuid.UUID) error {
	if err := fs.fileRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("services.DeleteFileMeta: failed to delete file metadata: %w", err)
	}
	return nil
}

func (fs *FileService) LookupFileMeta(ctx context.Context, id uuid.UUID) (*models.FileMeta, error) {
	fm, err := fs.fileRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("services.LookupFileMeta: failed to get file for id '%s': %w", id, err)
	}
	return fm, nil
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
