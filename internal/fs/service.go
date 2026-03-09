package fs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	_ "image/gif"
	_ "image/png"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type Service struct {
	meta MetaStore
	blob BlobStore
}

func NewService(m MetaStore, b BlobStore, dir string) *Service {
	return &Service{meta: m, blob: b}
}

func (s *Service) Upload(ctx context.Context, requests <-chan UploadRequest) <-chan UploadResult {
	results := make(chan UploadResult)

	go func() {
		for request := range requests {
			fileType := strings.Split(request.contentType, "/")[0]
			if fileType != "image" && fileType != "video" {
				results <- UploadResult{
					filename: request.filename,
					err:      ErrUnsupportedFileType,
				}
				continue
			}

			dbReadCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			if _, err := s.meta.Get(dbReadCtx, request.filename, request.userId); err == nil {
				results <- UploadResult{
					filename: request.filename,
					err:      ErrFileExists,
				}
				continue
			}

			meta := Metadata{
				Id:        uuid.New().String(),
				Filename:  request.filename,
				Thumbname: "thumb-" + request.filename,
				UserId:    request.userId,
			}

			err := func() error {
				f, err := stageFile(ctx, meta.Filename, request.reader)
				if err != nil {
					return fmt.Errorf("stage file to disk: %w", err)
				}
				defer f.Close()
				defer os.Remove(f.Name())

				thumbReader, err := generateThumbnail(ctx, f.Name())
				if err != nil {
					return fmt.Errorf("generate thumbnail: %w", err)
				}

				g, ctx := errgroup.WithContext(ctx)
				g.Go(func() error {
					return s.blob.Upload(ctx, meta.Filename, request.bucket, f)
				})

				g.Go(func() error {
					return s.blob.Upload(ctx, meta.Thumbname, request.bucket, thumbReader)
				})

				return g.Wait()
			}()

			if err != nil {
				results <- UploadResult{
					filename: request.filename,
					err:      err,
				}
				continue
			}

			dbWriteCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			if err := s.meta.Save(dbWriteCtx, &meta); err != nil {
				results <- UploadResult{
					filename: request.filename,
					err:      fmt.Errorf("save metadata: %w", err),
				}
				continue
			}

			results <- UploadResult{
				filename: request.filename,
				err:      nil,
			}
		}
	}()

	return results
}

func (s *Service) Download(ctx context.Context, request DownloadRequest) (*DownloadResult, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	metadata, err := s.meta.Get(dbCtx, request.filename, request.userId)
	if err != nil {
		return nil, fmt.Errorf("get metadata: %w", err)
	}

	if metadata.UserId != request.userId {
		return nil, errors.New("unauthorized request")
	}

	attrs, reader, err := s.blob.Download(ctx, request.filename, request.bucket)
	if err != nil {
		if errors.Is(err, ErrBlobNotExist) {
			dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()

			if err := s.meta.Delete(dbCtx, request.filename, request.userId); err != nil {
				return nil, fmt.Errorf("delete metadata for orphaned object: %w")
			}
		}

		return nil, fmt.Errorf("download blob %q: %w", request.filename, err)
	}

	return &DownloadResult{
		reader:      reader,
		contentType: attrs.ContentType,
		size:        attrs.Size,
		timestamp:   attrs.Created,
	}, nil
}

func (s *Service) Delete(ctx context.Context, request DeleteRequest) error {
	if err := s.blob.Delete(ctx, request.filename, request.bucket); err != nil {
		return fmt.Errorf("delete blob: %w", err)
	}

	if err := s.meta.Delete(ctx, request.filename, request.userId); err != nil {
		return fmt.Errorf("delete metadata: %w", err)
	}

	return nil
}

func stageFile(ctx context.Context, name string, r io.Reader) (*os.File, error) {
	f, err := os.CreateTemp("", name)
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}

	_, err = io.Copy(f, r)
	if err != nil {
		return nil, fmt.Errorf("copy to temp file: %w", err)
	}

	if _, err := f.Seek(0, 0); err != nil {
		f.Close()
		os.Remove(f.Name())
		return nil, fmt.Errorf("seek to start: %w", err)
	}

	return f, nil
}

func generateThumbnail(ctx context.Context, path string) (*bytes.Buffer, error) {
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
