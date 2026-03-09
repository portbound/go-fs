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
	Download(ctx context.Context, name, bucket string) (*storage.ObjectAttrs, *storage.Reader, error)
	Delete(ctx context.Context, name, bucket string) error
}

type MetaStore interface {
	Save(ctx context.Context, meta *Metadata) error
	Get(ctx context.Context, fileId, userId string) (*Metadata, error)
	GetAll(ctx context.Context, userId string) ([]*Metadata, error)
	Delete(ctx context.Context, fileId, userId string) error
}

type Metadata struct {
	Id        string `json:"id"`
	Filename  string `json:"filename"`
	Thumbname string `json:"thumbname"`
	UserId    string `json:"user_id"`
}

type UploadRequest struct {
	reader      io.ReadCloser
	filename    string
	contentType string
	userId      string
	bucket      string
}

type UploadResult struct {
	filename string
	err      error
}

type DownloadRequest struct {
	fileId string
	userId string
	bucket string
}

type DownloadResult struct {
	reader      io.ReadCloser
	contentType string
	size        int64
	timestamp   time.Time
}

type DeleteRequest struct {
	fileId string
	userId string
	bucket string
}

var (
	ErrFileExists          = errors.New("file already exists")
	ErrOrphanedFile        = errors.New("CRITICAL - orphaned file")
	ErrMediaNotExist       = errors.New("file not found in storage")
	ErrMediaCorrupted      = errors.New("one or more parts of the file are missing/corrupted")
	ErrUserUnauthorzied    = errors.New("user ")
	ErrUnsupportedFileType = errors.New("unsupported file type")
)
