package services_test

import (
	"context"
	_ "embed"
	"io"
	"time"
)

// // go:embed test.jpg
// var testFile []byte

type MockFileStore struct{}

func NewMockFileStore() *MockFileStore {
	return &MockFileStore{}
}

func (s *MockFileStore) Upload(ctx context.Context, fileName string, bucket string, src io.Reader) (int64, time.Time, error) {
	return 0, time.Time{}, nil
}

func (s *MockFileStore) Download(ctx context.Context, fileName string, bucket string) (io.ReadCloser, error) {
	return nil, nil
}

func (s *MockFileStore) Delete(ctx context.Context, fileName string, bucket string) error {
	return nil
}
