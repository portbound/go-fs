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
	meta  MetaStore
	media MediaStore
}

func NewService(meta MetaStore, media MediaStore) *Service {
	return &Service{meta: meta, media: media}
}

func (s *Service) Upload(ctx context.Context, requests <-chan UploadRequest) <-chan UploadResult {
	results := make(chan UploadResult)

	go func() {
		for request := range requests {
			fileType := strings.Split(request.ContentType, "/")[0]
			if fileType != "image" && fileType != "video" {
				results <- UploadResult{
					Filename: request.Filename,
					Err:      ErrUnsupportedFileType,
				}
				continue
			}

			dbReadCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			if _, err := s.meta.Get(dbReadCtx, request.Filename, request.UserId); err == nil {
				results <- UploadResult{
					Filename: request.Filename,
					Err:      ErrFileExists,
				}
				continue
			}

			meta := Metadata{
				Id:        uuid.New().String(),
				Filename:  request.Filename,
				Thumbname: "thumb-" + request.Filename,
				UserId:    request.UserId,
			}

			err := func() error {
				f, err := stageFile(meta.Filename, request.Reader)
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
					return s.media.Upload(ctx, meta.Filename, request.Bucket, f)
				})

				g.Go(func() error {
					return s.media.Upload(ctx, meta.Thumbname, request.Bucket, thumbReader)
				})

				return g.Wait()
			}()

			if err != nil {
				results <- UploadResult{
					Filename: request.Filename,
					Err:      err,
				}
				continue
			}

			dbWriteCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			if err := s.meta.Save(dbWriteCtx, &meta); err != nil {
				results <- UploadResult{
					Filename: request.Filename,
					Err:      fmt.Errorf("save metadata: %w", err),
				}
				continue
			}

			results <- UploadResult{
				Filename: request.Filename,
				Err:      nil,
			}
		}
	}()

	return results
}

func (s *Service) Download(ctx context.Context, request DownloadRequest) (*DownloadResult, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	metadata, err := s.meta.Get(dbCtx, request.FileId, request.UserId)
	if err != nil {
		return nil, fmt.Errorf("get metadata: %w", err)
	}

	if metadata.UserId != request.UserId {
		return nil, errors.New("unauthorized request")
	}

	attrs, reader, err := s.media.Download(ctx, request.FileId, request.Bucket)
	if err != nil {
		if errors.Is(err, ErrMediaNotExist) {
			return nil, ErrMediaCorrupted
		}

		return nil, fmt.Errorf("download media %q: %w", request.FileId, err)
	}

	return &DownloadResult{
		Reader:      reader,
		ContentType: attrs.ContentType,
		Size:        attrs.Size,
		Timestamp:   attrs.Created,
	}, nil
}

func (s *Service) GetMetadata(ctx context.Context, userId string) ([]Metadata, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return s.meta.GetAll(dbCtx, userId)
}

func (s *Service) Delete(ctx context.Context, request DeleteRequest) error {
	if err := s.media.Delete(ctx, request.FileId, request.Bucket); err != nil {
		return fmt.Errorf("delete media: %w", err)
	}

	if err := s.meta.Delete(ctx, request.FileId, request.UserId); err != nil {
		return fmt.Errorf("delete metadata: %w", err)
	}

	return nil
}

func stageFile(name string, r io.Reader) (*os.File, error) {
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
