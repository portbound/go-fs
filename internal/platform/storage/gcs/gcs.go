package gcs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"github.com/portbound/go-fs/internal/fs"
)

type Gcs struct {
	client    *storage.Client
	projectID string
	mu        sync.Mutex
}

func New(projectID string) (*Gcs, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx, storage.WithJSONReads())
	if err != nil {
		return nil, err
	}
	return &Gcs{client: client, projectID: projectID, mu: sync.Mutex{}}, nil
}

func (g *Gcs) Upload(ctx context.Context, name, bucket string, src io.Reader) error {
	bkt := g.client.Bucket(bucket)
	g.mu.Lock()
	_, err := bkt.Attrs(ctx)
	if err != nil {
		if !errors.Is(err, storage.ErrBucketNotExist) {
			g.mu.Unlock()
			return fmt.Errorf("get bucket %q attrs: %w", bucket, err)
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
			return fmt.Errorf("create bucket %q: %w", bucket, err)
		}
	}
	g.mu.Unlock()

	obj := g.client.Bucket(bucket).Object(name)

	w := obj.NewWriter(ctx)
	defer w.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*180)
	defer cancel()

	if _, err := io.Copy(w, src); err != nil {
		return fmt.Errorf("stream to bucket %q: %w", bucket, err)
	}

	return nil
}

func (g *Gcs) Download(ctx context.Context, name string, bucket string) (*storage.ObjectAttrs, *storage.Reader, error) {
	obj := g.client.Bucket(bucket).Object(name)

	attrs, err := obj.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, nil, fs.ErrMediaNotExist
		}
		return nil, nil, fmt.Errorf("get file attrs: %w", err)
	}

	r, err := obj.NewReader(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("new file reader: %w", err)
	}

	return attrs, r, nil
}

func (g *Gcs) Delete(ctx context.Context, name string, bucket string) error {
	obj := g.client.Bucket(bucket).Object(name)

	if err := obj.Delete(ctx); err != nil {
		return err
	}

	return nil
}
