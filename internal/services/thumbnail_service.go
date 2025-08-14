package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	_ "image/gif"
	_ "image/png"

	"github.com/portbound/go-fs/internal/models"
)

type ThumbnailService struct{}

func NewThumbnailService() *ThumbnailService {
	return &ThumbnailService{}
}

func (ts *ThumbnailService) Generate(ctx context.Context, fm *models.FileMeta) (io.Reader, error) {
	thumbPath := fmt.Sprintf("./local/tmp/thumb-%s.jpg", fm.Name)

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", fm.TmpFilePath,
		"-vf", "scale=150:-1",
		"-vframes", "1",
		thumbPath,
	)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg failed: %w", err)
	}

	file, err := os.Open(thumbPath)
	if err != nil {
		return nil, fmt.Errorf("could not open temp thumbnail file: %w", err)
	}
	defer file.Close()
	defer os.Remove(thumbPath)

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, file); err != nil {
		return nil, fmt.Errorf("services.Generate: failed to copy bytes for thumbnail: %w", err)
	}

	return buf, nil
}
