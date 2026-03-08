package fs

import (
	"context"
	"io"
)

type BlobStore interface {
	Upload(ctx context.Context, name string, bucket string, src io.Reader) (int64, int64, error)
	Download(ctx context.Context, name string, bucket string) (io.ReadCloser, error)
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
	ContentType   string `json:"type"`
	Size          int64  `json:"size"`
	Timestamp     int64  `json:"timestamp"`
	UserId        string `json:"userId"`
	DeletedAt     string `json:"deletedAt"`
}
