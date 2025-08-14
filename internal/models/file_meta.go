package models

type FileMeta struct {
	ID          string `json:"id"`
	ParentID    string `json:"parent-id"`
	ThumbID     string `json:"thumb-id"`
	Name        string `json:"name"`
	Owner       string `json:"owner"`
	ContentType string `json:"type"`
	TmpFilePath string `json:"tmp-file"`
}
