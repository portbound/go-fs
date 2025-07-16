package models

import (
	"time"

	"github.com/google/uuid"
)

type File struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Owner        string    `json:"owner"`
	Type         string    `json:"type"`
	Size         int       `json:"size"`
	Unit         string    `json:"unit"`
	UploadDate   time.Time `json:"upload-date"`
	ModifiedDate time.Time `json:"modified-date"`
	StoragePath  string    `json:"path"`
}
