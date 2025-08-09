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

	"golang.org/x/image/draw"
)

func CreateThumbnail(ctx context.Context, path string) (io.Reader, error) {
	file, err := os.Open(path)
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
}
