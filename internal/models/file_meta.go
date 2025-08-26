package models

import "time"

type FileMeta struct {
	ID          string    `json:"id"`
	ParentID    string    `json:"parent-id"`
	ThumbID     string    `json:"thumb-id"`
	Name        string    `json:"name"`
	ContentType string    `json:"type"`
	Size        int64     `json:"size"`
	UploadDate  time.Time `json:"upload-date"`
	Owner       string    `json:"owner"`
	TmpFilePath string    `json:"tmp-file"`
}
