// Package gcs
package gcs

import (
	"context"
	"errors"
	"io"
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

func (s *Storage) Upload(ctx context.Context, fileName string, bucketName string, src io.Reader) (int64, time.Time, error) {
	bkt := s.client.Bucket(bucketName)
	_, err := bkt.Attrs(ctx)
	if err != nil {
		if !errors.Is(err, storage.ErrBucketNotExist) {
			return 0, time.Time{}, err
		}

		// BUG: if a user does not have a bucket yet, and they try to upload several attachments, a race condition will occur since we're handling uploads concurrently. The first go routine to reach this method will start creating the bucket. If the other go routines are fast enough, they'll reach this logic before the bucket has been created, so the bkt.Attrs() check fails, but slow enough that they will get a conflict if they try to create the bucket themselves.

		// I guess we'll either A. need to create the bucket ourselves when we create the user, or B. create it somewhere earlier in the pipeline.

		// We could always look into a proper sign up process and see if there's a way to maybe send magic links? idk

		// attrs := &storage.BucketAttrs{
		// 	UniformBucketLevelAccess: storage.UniformBucketLevelAccess{
		// 		Enabled: true,
		// 	},
		// 	PublicAccessPrevention: 1,
		// 	Location:               "us-east4",
		// 	LocationType:           "Region",
		// }
		//
		// if err := bkt.Create(ctx, s.projectID, attrs); err != nil {
		// 	return 0, time.Time{}, err
		// }
	}

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
