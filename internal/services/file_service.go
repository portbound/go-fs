package services

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/portbound/go-fs/internal/models"
	"github.com/portbound/go-fs/internal/repositories"
	"golang.org/x/net/context"
)

type copyResult struct {
	bytes int64
	err   error
}

type FileService struct {
	repo             repositories.FileRepository
	localStoragePath string
}

func NewFileService(repo repositories.FileRepository, localStoragePath string) *FileService {
	return &FileService{repo: repo, localStoragePath: localStoragePath}
}

func (fs *FileService) StageFileToDisk(ctx context.Context, metadata *models.FileMeta, reader io.Reader) error {
	tmpDir := filepath.Join(fs.localStoragePath, "tmp")

	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("failed to create tmp storage dir: %w", err)
	}

	metadata.ID = uuid.New()
	tmpFileName := fmt.Sprintf("%s-%s", metadata.ID.String(), metadata.Name)
	tmpFilePath := filepath.Join(tmpDir, tmpFileName)

	tmpFile, err := os.Create(tmpFilePath)
	if err != nil {
		return fmt.Errorf("failed to create tmp file: %w", err)
	}
	defer tmpFile.Close()

	copyChan := make(chan copyResult, 1)
	go func() {
		bytes, err := io.Copy(tmpFile, reader)
		copyChan <- copyResult{bytes: bytes, err: err}
	}()

	select {
	case <-ctx.Done():
		os.Remove(tmpFilePath)
		return ctx.Err()
	case result := <-copyChan:
		if result.err != nil {
			os.Remove(tmpFilePath)
			return fmt.Errorf("failed to write to tmp file: %w", err)
		}
		metadata.Size = result.bytes
	}

	return nil
}
