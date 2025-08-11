// Package gcs
package gcs

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
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

func (g *GCSStorage) Upload(ctx context.Context, name string, path string) error {
	src, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("gcs.Upload: failed to open file %s: %w", path, err)
	}
	defer src.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*180)
	defer cancel()

	// fileName := filepath.Base(fm.TmpFilePath)
	obj := g.client.Bucket(g.bkt).Object(name)
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

func (g *GCSStorage) Download(ctx context.Context, fileName string) (io.ReadCloser, error) {
	obj := g.client.Bucket(g.bkt).Object(fileName)

	r, err := obj.NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("gcs.Get: failed to downlod file %s: %w", fileName, err)
	}

	return r, nil
}

func (g *GCSStorage) ListObjects(ctx context.Context, query *storage.Query) ([]string, error) {
	it := g.client.Bucket(g.bkt).Objects(ctx, query)
	files := []string{}
	for {
		attrs, err := it.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, fmt.Errorf("Bucket(%q).Objects: %w", g.bkt, err)
		}

		files = append(files, attrs.Name)
	}

	return files, nil
}

func (g *GCSStorage) Delete(ctx context.Context, id string) error {
	obj := g.client.Bucket(g.bkt).Object(id)

	if err := obj.Delete(ctx); err != nil {
		return fmt.Errorf("gcs.Delete: failed to delete %s: %w", id, err)
	}
	return nil
}
