package services_test

import (
	"context"
	_ "embed"
	"io"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/portbound/go-fs/internal/models"
	"github.com/portbound/go-fs/internal/services"
)

//go:embed test.jpg
var testFile []byte

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

func Test_fileService_ProcessBatch(t *testing.T) {
	tempDir := t.TempDir()

	path := filepath.Join(tempDir, "test.jpg")
	if err := os.WriteFile(path, testFile, 0644); err != nil {
	}

	tests := []struct {
		name  string
		batch []*models.FileMeta
		owner *models.User
		want  []error
	}{
		{
			name: "happy path",
			batch: []*models.FileMeta{
				{ID: "1", Name: "test.jpg", ContentType: "image/jpeg", Owner: "test@example.com", TmpFilePath: path},
				{ID: "2", Name: "test2.jpg", ContentType: "image/jpeg", Owner: "test@example.com", TmpFilePath: path},
				{ID: "3", Name: "test3.jpg", ContentType: "image/jpeg", Owner: "test@example.com", TmpFilePath: path},
			},
			owner: &models.User{
				ID:         "1",
				Email:      "test@example.com",
				BucketName: "testBucket",
			},
			want: []error{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFileStore := NewMockFileStore()
			mockFileMetaRepo := NewMockFileMetaRepository()
			mockFileMetaService := services.NewFileMetaService(mockFileMetaRepo)
			fileService := services.NewFileService(mockFileStore, mockFileMetaService, tempDir)

			got := fileService.ProcessBatch(context.Background(), tt.batch, tt.owner)
			if !slices.Equal(got, tt.want) {
				t.Errorf("ProcessBatch() = %v, want %v", got, tt.want)
			}
		})
	}
}
