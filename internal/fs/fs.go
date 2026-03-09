package fs

import (
	"context"
	"io"
	"time"

	"cloud.google.com/go/storage"
)

type BlobStore interface {
	Upload(ctx context.Context, name string, bucket string, src io.Reader) (int64, int64, error)
	Download(ctx context.Context, name string, bucket string) (*storage.ObjectAttrs, *storage.Reader, error)
	Delete(ctx context.Context, name string, bucket string) error
}

type MetaStore interface {
	Save(ctx context.Context, meta *Metadata) error
	Get(ctx context.Context, fileId, userId string) (*Metadata, error)
	Delete(ctx context.Context, fileId, userId string) error
}

type Metadata struct {
	Id            string `json:"id"`
	FileName      string `json:"fileName"`
	ThumbnailName string `json:"thumbnailName"`
	UserId        string `json:"userId"`
	DeletedAt     string `json:"deletedAt"`
}

type UploadRequest struct {
	path        string
	filename    string
	contentType string
}

type UploadResult struct {
	filename string
	err      error
}

type DownloadResult struct {
	reader      io.Reader
	contentType string
	size        int64
	timestamp   time.Time
}
