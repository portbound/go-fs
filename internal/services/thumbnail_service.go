package services

import (
	"context"
	"fmt"
	"image"
	"io"
	"os"

	_ "image/gif"
	"image/jpeg"
	_ "image/png"

	"github.com/portbound/go-fs/internal/models"
	"golang.org/x/image/draw"
)

type ThumbnailGenerator struct {
	dir    string
	width  int
	height int
}

func NewThumbnailGenerator(path string) *ThumbnailGenerator {
	return &ThumbnailGenerator{dir: path, width: 150, height: 150}
}

func (tg *ThumbnailGenerator) CreateThumbnail(ctx context.Context, fm *models.FileMeta) (io.Reader, error) {
	file, err := os.Open(fm.TmpFilePath)
	if err != nil {
		return nil, fmt.Errorf("services.CreateThumbnail: failed to open file: %w", err)
	}
	defer file.Close()

	srcImg, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("services.CreateThumbnail: failed to decode source image: %w", err)
	}

	thumbnail := image.NewRGBA(image.Rect(0, 0, tg.width, tg.height))
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
}
