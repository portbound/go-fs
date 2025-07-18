package models

import (
	"time"

	"github.com/google/uuid"
)

type File struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	OriginalName string    `json:"original-name"`
	Owner        string    `json:"owner"`
	Type         string    `json:"type"`
	Size         int64     `json:"size"`
	UploadDate   time.Time `json:"upload-date"`
	ModifiedDate time.Time `json:"modified-date"`
	StoragePath  string    `json:"path"`
}
