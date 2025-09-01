package models

import "time"

type FileMeta struct {
	ID          string    `json:"id"`
	ParentID    string    `json:"parentId"`
	ThumbID     string    `json:"thumbId"`
	Name        string    `json:"name"`
	ContentType string    `json:"type"`
	Size        int64     `json:"size"`
	UploadDate  time.Time `json:"uploadDate"`
	Owner       string    `json:"owner"`
	TmpFilePath string    `json:"tmpFile"`
}
