package fs

import (
	"context"
	"errors"
	"io"
	"time"

	"cloud.google.com/go/storage"
)

type MediaStore interface {
	Upload(ctx context.Context, name, bucket string, src io.Reader) error
	// TODO: need to come up with a better signature for this since these return vals are CGS specific
	Download(ctx context.Context, name, bucket string) (*storage.ObjectAttrs, *storage.Reader, error)
	Delete(ctx context.Context, name, bucket string) error
}

type MetaStore interface {
	Save(ctx context.Context, meta *Metadata) error
	Get(ctx context.Context, fileId, userId string) (*Metadata, error)
	GetAll(ctx context.Context, userId string) ([]Metadata, error)
	Delete(ctx context.Context, fileId, userId string) error
}

type Metadata struct {
	Id        string `json:"id"`
	Filename  string `json:"filename"`
	Thumbname string `json:"thumbname"`
	UserId    string `json:"user_id"`
}

type UploadRequest struct {
	Reader      io.ReadCloser
	Filename    string
	ContentType string
	UserId      string
	Bucket      string
}

type UploadResult struct {
	Filename string
	Err      error
}

type DownloadRequest struct {
	FileId string
	UserId string
	Bucket string
}

type DownloadResult struct {
	Reader      io.ReadCloser
	ContentType string
	Size        int64
	Timestamp   time.Time
}

type DeleteRequest struct {
	FileId string
	UserId string
	Bucket string
}

var (
	ErrFileExists          = errors.New("file already exists")
	ErrOrphanedFile        = errors.New("CRITICAL - orphaned file")
	ErrMediaNotExist       = errors.New("file not found in storage")
	ErrMediaCorrupted      = errors.New("one or more parts of the file are missing/corrupted")
	ErrUserUnauthorzied    = errors.New("user ")
	ErrUnsupportedFileType = errors.New("unsupported file type")
)
