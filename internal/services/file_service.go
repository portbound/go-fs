package services

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/portbound/go-fs/internal/models"
	"github.com/portbound/go-fs/internal/repositories"
)

type FileService struct {
	repo repositories.FileRepository
}

func NewFileService(repo repositories.FileRepository) *FileService {
	return &FileService{repo: repo}
}

func (fs *FileService) UploadFile(r io.Reader, metadata *models.FileMetadata) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get cwd: %w", err)
	}

	tmpStorage := filepath.Join(cwd, "tmp")
	if err := os.MkdirAll(tmpStorage, 0755); err != nil {
		return fmt.Errorf("failed to create tmp storage dir: %w", err)
	}

	filename := fmt.Sprintf("%s-%s", uuid.New().String(), metadata.Name)
	tmpPath := filepath.Join(tmpStorage, filename)

	outFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create tmp file: %w", err)
	}
	defer outFile.Close()

	bytes, err := io.Copy(outFile, r)
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write to tmp file: %w", err)
	}

	metadata.Size = bytes
	return nil
	// _, err := streamToDisk(r, metadata)
	// if err != nil {
	// 	return err
	// }
	// return nil
}

func uploadToCloud(tmpPath string, metadata *models.FileMetadata) {
	panic("unimplemented")
}

func streamToDisk(r io.Reader, metadata *models.FileMetadata) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get cwd: %w", err)
	}

	tmpStorage := filepath.Join(cwd, "tmp")
	if err := os.MkdirAll(tmpStorage, 0755); err != nil {
		return "", fmt.Errorf("failed to create tmp storage dir: %w", err)
	}

	filename := fmt.Sprintf("%s-%s", uuid.New().String(), metadata.Name)
	tmpPath := filepath.Join(tmpStorage, filename)

	outFile, err := os.Create(tmpPath)
	if err != nil {
		return "", fmt.Errorf("failed to create tmp file: %w", err)
	}
	defer outFile.Close()

	bytes, err := io.Copy(outFile, r)
	if err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("failed to write to tmp file: %w", err)
	}

	metadata.Size = bytes
	return tmpPath, nil
}
