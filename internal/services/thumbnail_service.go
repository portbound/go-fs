package services

import (
	"fmt"
	"image"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/portbound/go-fs/internal/models"
	"github.com/portbound/go-fs/internal/util"
	"golang.org/x/image/draw"
)

type Generator struct {
	dir    string
	width  int
	height int
}

func NewThumbnailGenerator(path string) (*Generator, error) {
	return &Generator{dir: path, width: 150, height: 150}, nil
}

func (g *Generator) GenerateThumbnailFromImage(fm *models.FileMeta) error {
	inputFile, err := os.Open(filepath.Join(g.dir, fm.Name))
	if err != nil {
		return fmt.Errorf("services.GenerateThumbnailFromImage: failed to open file: %w", err)
	}
	defer inputFile.Close()

	img, _, err := image.Decode(inputFile)
	if err != nil {
		return fmt.Errorf("services.GenerateThumbnailFromImage: failed to decode image: %w", err)
	}

	thumbnail := image.NewRGBA(image.Rect(0, 0, g.width, g.height))

	draw.CatmullRom.Scale(thumbnail, thumbnail.Bounds(), img, img.Bounds(), draw.Src, nil)

	outputFile, err := os.Create(path.Join(g.dir, fm.Name, "_thumbnail"))
	if err != nil {
		return fmt.Errorf("services.GenerateThumbnailFromImage: failed to create thumbnail: %w", err)
	}
	defer outputFile.Close()

	imgType := strings.TrimPrefix(fm.ContentType, "image/")

	switch imgType {
	case "jpg", "png", "gif":
		thumbnail, _, err := image.Decode(outputFile)
		if err != nil {
			fmt.Errorf("services.GenerateThumbnailFromImage: failed to decode thumbnail: %w", err)
		}

	// TODO: create pipe
	default:
		fmt.Errorf("services.GenerateThumbnailFromImage: unsupported image type: %s", imgType)
	}
	return nil
}

func (g *Generator) GenerateThumbnailFromVideo(name string) (*models.Thumbnail, error) {
	return nil, nil
}
