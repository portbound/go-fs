package models

import (
	"github.com/google/uuid"
)

type FileMeta struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Owner         string    `json:"owner"`
	ContentType   string    `json:"type"`
	Size          int64     `json:"size"`
	OriginalPath  string    `json:"original-path"`
	ThumbnailPath string    `json:"thumbnail-path"`
	PreviewPath   string    `json:"preview-path"`
	TmpDir        string    `json:"tmp-dir"`
}
