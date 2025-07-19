package models

import (
	"time"

	"github.com/google/uuid"
)

type FileMeta struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Owner       string    `json:"owner"`
	Type        string    `json:"type"`
	Size        int64     `json:"size"`
	UploadDate  time.Time `json:"upload-date"`
	StoragePath string    `json:"path"`
}
