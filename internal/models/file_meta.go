package models

import (
	"github.com/google/uuid"
)

type FileMeta struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Owner        string    `json:"owner"`
	ContentType  string    `json:"type"`
	Size         int64     `json:"size"`
	FilePath     string    `json:"file-path"`
	TmpFilePath  string    `json:"tmp-file"`
	Thumbnail    string    `json:"thumbnail-path"`
	TmpThumbPath string    `json:"tmp-file-thumb"`
	PreviewPath  string    `json:"preview-path"`
}
