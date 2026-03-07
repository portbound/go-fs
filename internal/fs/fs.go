package fs

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"time"

	_ "image/gif"
	_ "image/png"

	"cloud.google.com/go/storage"
	"github.com/google/uuid"
	"github.com/portbound/go-fs/internal/user"
)

type MetaStore interface {
	Save(ctx context.Context, meta *Metadata) error
	Get(ctx context.Context, fileId, userId string) (*Metadata, error)
	Delete(ctx context.Context, fileId, userId string) error
}

type BlobStore interface {
	Upload(ctx context.Context, name string, bucket string, src io.Reader) (int64, int64, error)
	Download(ctx context.Context, name string, bucket string) (io.ReadCloser, error)
	Delete(ctx context.Context, name string, bucket string) error
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

type Service struct {
	meta   MetaStore
	blob   BlobStore
	tmpDir string
}

var (
	ErrFileExists   = errors.New("file already exists")
	ErrOrphanedFile = errors.New("CRITICAL - orphaned file")
)

func NewService(m MetaStore, b BlobStore, dir string) *Service {
	return &Service{meta: m, blob: b, tmpDir: dir}
}

func (s *Service) Upload(ctx context.Context, reader io.Reader, name, contentType string, owner *user.User) error {
	dbReadCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	_, err := s.meta.Get(dbReadCtx, name, owner.Id)
	if err == nil {
		return ErrFileExists
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	meta := Metadata{
		Id:            uuid.New().String(),
		FileName:      name,
		ThumbnailName: "thumb-" + name,
		ContentType:   contentType,
		UserId:        owner.Id,
	}

	dbWriteCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := s.meta.Save(dbWriteCtx, &meta); err != nil {
		return fmt.Errorf("save metadata: %w", err)
	}

	size, timestamp, err := s.blob.Upload(ctx, meta.FileName, owner.Bucket, reader)
	if err != nil {
		return fmt.Errorf("upload %q: %w", meta.FileName, err)
	}
	meta.Size = size
	meta.Timestamp = timestamp

	thumbReader, err := generateThumbnail(ctx, s.tmpDir)
	if err != nil {
		return fmt.Errorf("generate thumbnail: %w", err)
	}

	_, _, err = s.blob.Upload(ctx, meta.ThumbnailName, owner.Bucket, thumbReader)
	if err != nil {
		return fmt.Errorf("upload thumbnail %q: %w", meta.ThumbnailName, err)
	}

	dbWriteCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := s.meta.Save(dbWriteCtx, &meta); err != nil {
		saveErr := errors.Join(err)
		if err := s.blob.Delete(ctx, meta.FileName, owner.Bucket); err != nil {
			saveErr = errors.Join(fmt.Errorf("%v: delete %q: %w", ErrOrphanedFile, meta.FileName, err))
		}

		if err := s.blob.Delete(ctx, meta.ThumbnailName, owner.Bucket); err != nil {
			saveErr = errors.Join(fmt.Errorf("%v: delete %q: %w", ErrOrphanedFile, meta.ThumbnailName, err))
		}

		return fmt.Errorf("save file metadata: %w", saveErr)
	}

	return nil
}

// func (s *Service) Upload(ctx context.Context, reader io.Reader, name, contentType string, owner *user.User) error {
// 	meta := Metadata{
// 		Id:            uuid.New().String(),
// 		FileName:      name,
// 		ThumbnailName: "thumb-" + name,
// 		ContentType:   contentType,
// 		Email:         owner.Email,
// 	}
//
// 	dbReadCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
// 	defer cancel()
//
// 	_, err := s.meta.Get(dbReadCtx, meta.FileName, owner)
// 	if err == nil {
// 		return ErrFileExists
// 	}
// 	if !errors.Is(err, sql.ErrNoRows) {
// 		return err
// 	}
//
// 	size, timestamp, err := s.blob.Upload(ctx, meta.FileName, owner.Bucket, reader)
// 	if err != nil {
// 		return fmt.Errorf("upload %q: %w", meta.FileName, err)
// 	}
// 	meta.Size = size
// 	meta.Timestamp = timestamp
//
// 	thumbReader, err := generateThumbnail(ctx, s.tmpDir)
// 	if err != nil {
// 		return fmt.Errorf("generate thumbnail: %w", err)
// 	}
//
// 	_, _, err = s.blob.Upload(ctx, meta.ThumbnailName, owner.Bucket, thumbReader)
// 	if err != nil {
// 		return fmt.Errorf("upload thumbnail %q: %w", meta.ThumbnailName, err)
// 	}
//
// 	dbWriteCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
// 	defer cancel()
// 	if err := s.meta.Save(dbWriteCtx, &meta); err != nil {
// 		saveErr := errors.Join(err)
// 		if err := s.blob.Delete(ctx, meta.FileName, owner.Bucket); err != nil {
// 			saveErr = errors.Join(fmt.Errorf("%v: delete %q: %w", ErrOrphanedFile, meta.FileName, err))
// 		}
//
// 		if err := s.blob.Delete(ctx, meta.ThumbnailName, owner.Bucket); err != nil {
// 			saveErr = errors.Join(fmt.Errorf("%v: delete %q: %w", ErrOrphanedFile, meta.ThumbnailName, err))
// 		}
//
// 		return fmt.Errorf("save file metadata: %w", saveErr)
// 	}
//
// 	return nil
// }

func (s *Service) Download(ctx context.Context, id string, owner *user.User) (io.ReadCloser, error) {
	gcsReader, err := s.blob.Download(ctx, id, owner.Bucket)
	if err != nil {
		// TODO: need to check to see if the error returned was ErrObjNotExist or whatever from the storage package so we can delete it from here if the client tries to download a file that doesn't exist
		return nil, err
	}

	return gcsReader, nil
}

func (s *Service) Delete(ctx context.Context, id string, owner *user.User) error {
	if err := s.blob.Delete(ctx, id, owner.Bucket); err != nil {
		return fmt.Errorf("delete: %w")
	}

	if err := s.meta.Delete(ctx, id, owner.Id); err != nil {
		deleteErr := errors.Join(err)
		return fmt.Errorf("delete: %w", err)
	}

	return nil
}

func generateThumbnail(ctx context.Context, path string) (io.Reader, error) {
	args := []string{
		"-i", path,
		"-vf", "scale=150:150:force_original_aspect_ratio=increase,crop=150:150",
		"-vframes", "1",
		"-f", "mjpeg",
		"-",
	}

	ffmpegCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var buf bytes.Buffer
	cmd := exec.CommandContext(ffmpegCtx, "ffmpeg", args...)
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return &buf, nil
}
