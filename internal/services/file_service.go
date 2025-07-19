package services

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/portbound/go-fs/internal/models"
	"github.com/portbound/go-fs/internal/repositories"
)

type FileService struct {
	repo             repositories.FileRepository
	localStoragePath string
}

func NewFileService(repo repositories.FileRepository, localStoragePath string) *FileService {
	return &FileService{repo: repo, localStoragePath: localStoragePath}
}

func (fs *FileService) UploadFile(r io.Reader, metadata *models.FileMetadata) error {
	metadata.ID = uuid.New()

	_, bytes, err := streamToDisk(r, metadata, fs.localStoragePath)
	if err != nil {
		return err
	}

	// TODO: implement streamToCloud()
	// Streaming to the disk from the client first
	// We don't want to funnel from client => bucket directly since that will keep the client waiting longer than they need to
	// We can stream from /tmpdir => bucket instead, and after successful upload to cloud delete the asset from /tmpdir

	metadata.UploadDate = time.Now()
	metadata.Size = bytes
	return nil
}

func streamToDisk(r io.Reader, metadata *models.FileMetadata, localStoragePath string) (string, int64, error) {
	tmpDir := filepath.Join(localStoragePath, "tmp")

	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", 0, fmt.Errorf("failed to create tmp storage dir: %w", err)
	}

	tmpFileName := fmt.Sprintf("%s-%s", metadata.ID.String(), metadata.Name)
	tmpFilePath := filepath.Join(tmpDir, tmpFileName)

	tmpFile, err := os.Create(tmpFilePath)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create tmp file: %w", err)
	}
	defer tmpFile.Close()

	bytes, err := io.Copy(tmpFile, r)
	if err != nil {
		os.Remove(tmpFilePath)
		return "", 0, fmt.Errorf("failed to write to tmp file: %w", err)
	}

	return tmpFilePath, bytes, nil
}
