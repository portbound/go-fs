// Package gcs
package gcs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"cloud.google.com/go/storage"
	"github.com/portbound/go-fs/internal/models"
)

type GCSStorage struct {
	client *storage.Client
	bkt    string
}

func NewGCSStorage(ctx context.Context, bkt string) (*GCSStorage, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &GCSStorage{client: client, bkt: bkt}, nil
}

func (g *GCSStorage) Upload(ctx context.Context, path string) error {
	src, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("gcs.Upload: failed to open file %s: %w", path, err)
	}
	defer src.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*180)
	defer cancel()

	fileName := filepath.Base(path)
	obj := g.client.Bucket(g.bkt).Object(fileName)
	obj = obj.If(storage.Conditions{DoesNotExist: true})
	wc := obj.NewWriter(ctx)

	if _, err := io.Copy(wc, src); err != nil {
		return fmt.Errorf("gcs.Upload: failed to copy file to writer: %w", err)
	}

	if err := wc.Close(); err != nil {
		return fmt.Errorf("gcs.Upload: failed to commit upload: %w", err)
	}
	return nil
}

func (g *GCSStorage) Download(ctx context.Context, fm *models.FileMeta) (io.ReadCloser, error) {
	obj := g.client.Bucket(g.bkt).Object(fm.Name)

	r, err := obj.NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("gcs.Get: failed to create reader: %w", err)
	}

	return r, nil
}
func (g *GCSStorage) Delete(ctx context.Context, fileName string) error {
	obj := g.client.Bucket(g.bkt).Object(fileName)

	if err := obj.Delete(ctx); err != nil {
		return fmt.Errorf("gcs.Delete: failed to delete %s: %w", fileName, err)
	}
	return nil
}
