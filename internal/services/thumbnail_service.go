package services

import (
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"os/exec"
	"strings"

	_ "image/gif"
	"image/jpeg"
	_ "image/png"

	"github.com/portbound/go-fs/internal/models"
	"golang.org/x/image/draw"
)

type ThumbnailService struct{}

func NewThumbnailService() *ThumbnailService {
	return &ThumbnailService{}
}

func (ts *ThumbnailService) Generate(ctx context.Context, fm *models.FileMeta) (io.Reader, error) {
	fileType := strings.ToLower(strings.Split(fm.ContentType, "/")[0])
	switch fileType {
	case "image":
		return ts.generateFromImage(ctx, fm)
	case "video":
		return ts.generateFromVideo(ctx, fm)
	default:
		return nil, nil
	}
}

func (ts *ThumbnailService) generateFromImage(ctx context.Context, fm *models.FileMeta) (io.Reader, error) {
	fileSubType := strings.ToLower(strings.Split(fm.ContentType, "/")[1])
	switch fileSubType {
	case "jpg", "png", "gif":
		file, err := os.Open(fm.TmpFilePath)
		if err != nil {
			return nil, fmt.Errorf("services.CreateThumbnail: failed to open file: %w", err)
		}
		defer file.Close()

		srcImg, _, err := image.Decode(file)
		if err != nil {
			return nil, fmt.Errorf("services.CreateThumbnail: failed to decode source image: %w", err)
		}

		thumbnail := image.NewRGBA(image.Rect(0, 0, 150, 150))
		draw.CatmullRom.Scale(thumbnail, thumbnail.Bounds(), srcImg, srcImg.Bounds(), draw.Src, nil)

		pipeReader, pipeWriter := io.Pipe()
		go func() {
			defer pipeWriter.Close()
			err := jpeg.Encode(pipeWriter, thumbnail, nil)
			if err != nil {
				pipeWriter.CloseWithError(err)
			}
		}()

		return pipeReader, nil
	default:
		return nil, errors.New("services.ProcessBatch: failed to create thumnail for %s: file type not supported")
	}
}

func (ts *ThumbnailService) generateFromVideo(ctx context.Context, fm *models.FileMeta) (io.Reader, error) {
	thumbPath := fm.TmpFilePath + "-thumb.jpg"

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", fm.TmpFilePath,
		"-vf", "scale=150:-1", // Scale width to 150, maintain aspect ratio
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
	return file, nil
}
