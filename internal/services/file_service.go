package services

import "github.com/portbound/go-fs/internal/repositories"

type FileService struct {
	repo repositories.FileRepository
}

func NewFileService(repo repositories.FileRepository) *FileService {
	return &FileService{repo: repo}
}
