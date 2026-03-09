package gcs

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"cloud.google.com/go/storage"
)

type Gcs struct {
	client    *storage.Client
	projectID string
	mu        sync.Mutex
}

func New(projectID string) (*Gcs, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &Gcs{client: client, projectID: projectID, mu: sync.Mutex{}}, nil
}

func (g *Gcs) Upload(ctx context.Context, fileName string, bucketName string, src io.Reader) (int64, int64, error) {
	bkt := g.client.Bucket(bucketName)
	g.mu.Lock()
	_, err := bkt.Attrs(ctx)
	if err != nil {
		if !errors.Is(err, storage.ErrBucketNotExist) {
			g.mu.Unlock()
			return 0, 0, err
		}

		attrs := &storage.BucketAttrs{
			UniformBucketLevelAccess: storage.UniformBucketLevelAccess{
				Enabled: true,
			},
			PublicAccessPrevention: 1,
			Location:               "us-east4",
			LocationType:           "Region",
		}

		if err := bkt.Create(ctx, g.projectID, attrs); err != nil {
			g.mu.Unlock()
			return 0, 0, err
		}
	}
	g.mu.Unlock()

	obj := g.client.Bucket(bucketName).Object(fileName)
	wc := obj.NewWriter(ctx)

	ctx, cancel := context.WithTimeout(ctx, time.Second*180)
	defer cancel()

	if _, err := io.Copy(wc, src); err != nil {
		return 0, 0, err
	}

	if err := wc.Close(); err != nil {
		return 0, 0, err
	}

	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return 0, 0, err
	}

	return attrs.Size, attrs.Created.Unix(), nil
}

func (g *Gcs) Download(ctx context.Context, name string, bucket string) (*storage.ObjectAttrs, *storage.Reader, error) {
	obj := g.client.Bucket(bucket).Object(name)

	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return nil, nil, err
	}

	reader, err := obj.NewReader(ctx)
	if err != nil {
		return nil, nil, err
	}

	return attrs, reader, nil
}

func (g *Gcs) Delete(ctx context.Context, name string, bucket string) error {
	obj := g.client.Bucket(bucket).Object(name)

	if err := obj.Delete(ctx); err != nil {
		return err
	}

	return nil
}
