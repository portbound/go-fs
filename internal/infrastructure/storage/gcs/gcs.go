package gcs

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"cloud.google.com/go/storage"
	"github.com/portbound/go-fs/internal/models"
)

type GCSStorage struct {
	client *storage.Client
	bkt    string
}

func NewGCSStorage(bkt string) (*GCSStorage, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &GCSStorage{client: client, bkt: bkt}, nil
}

func (g *GCSStorage) Upload(ctx context.Context, fm *models.FileMeta) error {
	file, err := os.Open(fm.TmpDir)
	if err != nil {
		return fmt.Errorf("gcs.Upload: failed to open temp file %s: %w", fm.TmpDir, err)
	}
	defer file.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*180)
	defer cancel()

	obj := g.client.Bucket(g.bkt).Object(fm.Name)
	obj = obj.If(storage.Conditions{DoesNotExist: true})
	wc := obj.NewWriter(ctx)

	if _, err := io.Copy(wc, file); err != nil {
		return fmt.Errorf("gcs.Upload: failed to copy file to writer: %w", err)
	}

	if err := wc.Close(); err != nil {
		return fmt.Errorf("gcs.Upload: failed to commit upload: %w", err)
	}
	return nil
}

func (g *GCSStorage) Delete(ctx context.Context, fm *models.FileMeta) error {
	obj := g.client.Bucket(g.bkt).Object(fm.Name)

	if err := obj.Delete(ctx); err != nil {
		return fmt.Errorf("gcs.Delete: failed to delete file: %w", err)
	}
	return nil
}
