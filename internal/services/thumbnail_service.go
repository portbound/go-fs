package services

import (
	"bytes"
	"context"
	"io"
	"os/exec"
	"time"

	_ "image/gif"
	_ "image/png"

	"github.com/portbound/go-fs/internal/models"
)

func GenerateThumbnail(ctx context.Context, fm *models.FileMeta) (io.Reader, error) {
	var buf bytes.Buffer

	args := []string{
		"-i", fm.TmpFilePath,
		"-vf", "scale=150:150:force_original_aspect_ratio=increase,crop=150:150",
		"-vframes", "1",
		"-f", "mjpeg",
		"-",
	}

	ffmpegCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ffmpegCtx, "ffmpeg", args...)
	cmd.Stdout = &buf

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return &buf, nil
}
