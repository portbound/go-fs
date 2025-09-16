// Package gcs
package gcs

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"cloud.google.com/go/storage"
)

type Storage struct {
	client    *storage.Client
	projectID string
	mu        sync.Mutex
}

func NewStorage(ctx context.Context, projectID string) (*Storage, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &Storage{client: client, projectID: projectID, mu: sync.Mutex{}}, nil
}

func (s *Storage) Upload(ctx context.Context, fileName string, bucketName string, src io.Reader) (int64, time.Time, error) {
	bkt := s.client.Bucket(bucketName)
	s.mu.Lock()
	_, err := bkt.Attrs(ctx)
	if err != nil {
		if !errors.Is(err, storage.ErrBucketNotExist) {
			s.mu.Unlock()
			return 0, time.Time{}, err
		}

		attrs := &storage.BucketAttrs{
			UniformBucketLevelAccess: storage.UniformBucketLevelAccess{
				Enabled: true,
			},
			PublicAccessPrevention: 1,
			Location:               "us-east4",
			LocationType:           "Region",
		}

		if err := bkt.Create(ctx, s.projectID, attrs); err != nil {
			s.mu.Unlock()
			return 0, time.Time{}, err
		}
	}
	s.mu.Unlock()

	obj := s.client.Bucket(bucketName).Object(fileName)
	wc := obj.NewWriter(ctx)

	ctx, cancel := context.WithTimeout(ctx, time.Second*180)
	defer cancel()

	if _, err := io.Copy(wc, src); err != nil {
		return 0, time.Time{}, err
	}

	if err := wc.Close(); err != nil {
		return 0, time.Time{}, err
	}

	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return 0, time.Time{}, err
	}

	return attrs.Size, attrs.Created, nil
}

func (s *Storage) Download(ctx context.Context, fileName string, bucketName string) (io.ReadCloser, error) {
	obj := s.client.Bucket(bucketName).Object(fileName)

	r, err := obj.NewReader(ctx)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (s *Storage) Delete(ctx context.Context, id string, bucketName string) error {
	obj := s.client.Bucket(bucketName).Object(id)

	if err := obj.Delete(ctx); err != nil {
		return err
	}

	return nil
}
