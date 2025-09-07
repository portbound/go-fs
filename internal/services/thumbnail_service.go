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

func GenerateThumbnail(ctx context.Context, fm *models.FileMeta) (io.Reader, error) {
	thumbPath := fmt.Sprintf("./local/tmp/thumb-%s.jpg", fm.Name)

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", fm.TmpFilePath,
		"-vf", "scale=150:150:force_original_aspect_ratio=increase,crop=150:150",
		"-vframes", "1",
		thumbPath,
	)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("[GenerateThumbnail] ffmpeg cmd failed: %w", err)
	}

	file, err := os.Open(thumbPath)
	if err != nil {
		return nil, fmt.Errorf("[GenerateThumbnail] failed could not open tmp thumbnail file: %w", err)
	}
	defer file.Close()
	defer os.Remove(thumbPath)

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, file); err != nil {
		return nil, fmt.Errorf("[GenerateThumbnail] failed to copy bytes to thumbnail buffer: %w", err)
	}

	return buf, nil
}
