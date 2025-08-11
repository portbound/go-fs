package models

import (
	"github.com/google/uuid"
)

type FileMeta struct {
	ID           uuid.UUID `json:"id"`
	ThumbID      string    `json:"thumb-id"`
	Name         string    `json:"name"`
	Owner        string    `json:"owner"`
	ContentType  string    `json:"type"`
	TmpFilePath  string    `json:"tmp-file"`
	TmpThumbPath string    `json:"tmp-thumb"`
}
