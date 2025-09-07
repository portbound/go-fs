// Package gcs
package gcs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
)

type Storage struct {
	client    *storage.Client
	projectID string
}

func NewStorage(ctx context.Context, projectID string) (*Storage, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &Storage{client: client, projectID: projectID}, nil
}

func (s *Storage) Upload(ctx context.Context, fileName string, owner string, diskPath string) (int64, time.Time, error) {
	src, err := os.Open(diskPath)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("[gcs.Upload] failed to open file %s: %w", diskPath, err)
	}
	defer src.Close()

	bucketName := strings.Split(owner, "@")[0]
	bkt := s.client.Bucket(bucketName)
	_, err = bkt.Attrs(ctx)
	if errors.Is(storage.ErrBucketNotExist, err) {
		if err := bkt.Create(ctx, s.projectID, nil); err != nil {
			return 0, time.Time{}, fmt.Errorf("[gcs.Upload] failed to create new bucket %s: %w", bucketName, err)
		}
	} else {
		return 0, time.Time{}, fmt.Errorf("[gcs.Upload] failed to verify bucket attributes for %s: %w", bucketName, err)
	}

	obj := s.client.Bucket(bucketName).Object(fileName)
	wc := obj.NewWriter(ctx)

	ctx, cancel := context.WithTimeout(ctx, time.Second*180)
	defer cancel()

	if _, err := io.Copy(wc, src); err != nil {
		return 0, time.Time{}, fmt.Errorf("[gcs.Upload] failed to copy file to writer: %w", err)
	}

	if err := wc.Close(); err != nil {
		return 0, time.Time{}, fmt.Errorf("[gcs.Upload] failed to commit upload: %w", err)
	}

	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("[gcs.Upload] failed to get object attributes: %w", err)
	}

	return attrs.Size, attrs.Created, nil
}

func (s *Storage) Download(ctx context.Context, fileName string, owner string) (io.ReadCloser, error) {
	bucketName := strings.Split(owner, "@")[0]
	obj := s.client.Bucket(bucketName).Object(fileName)

	r, err := obj.NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("[gcs.Get] failed to downlod file %s: %w", fileName, err)
	}
	return r, nil
}

func (s *Storage) Delete(ctx context.Context, id string, owner string) error {
	bucketName := strings.Split(owner, "@")[0]
	obj := s.client.Bucket(bucketName).Object(id)

	if err := obj.Delete(ctx); err != nil {
		return fmt.Errorf("[gcs.Delete] failed to delete %s: %w", id, err)
	}

	return nil
}
