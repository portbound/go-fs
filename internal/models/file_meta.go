package models

import (
	"github.com/google/uuid"
)

type FileMeta struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Owner        string    `json:"owner"`
	ContentType  string    `json:"type"`
	FilePath     string    `json:"file-path"`
	TmpFilePath  string    `json:"tmp-file"`
	ThumbPath    string    `json:"thumb-path"`
	TmpThumbPath string    `json:"tmp-thumb"`
	PreviewPath  string    `json:"preview-path"`
}
