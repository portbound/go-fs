package services_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/portbound/go-fs/internal/infrastructure/database/sqlite"
	"github.com/portbound/go-fs/internal/models"
	"github.com/portbound/go-fs/internal/repositories"
	"github.com/portbound/go-fs/internal/services"
)

func TestFileService_UploadFile(t *testing.T) {
	tests := []struct {
		name     string
		repo     repositories.FileRepository
		r        io.Reader
		metadata *models.FileMetadata
		wantErr  bool
	}{
		{
			name: "passing",
			metadata: &models.FileMetadata{
				Name:  "test.txt",
				Type:  "text/polain",
				Owner: "test",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, buf := setup(t)
			tt.repo = repo
			tt.r = buf
			testLocalStoragePath := t.TempDir()
			fs := services.NewFileService(tt.repo, testLocalStoragePath)

			gotErr := fs.UploadFile(tt.r, tt.metadata)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("UploadFile() failed: %v", gotErr)
				}
				return
			}

			if tt.wantErr {
				t.Fatal("UploadFile() succeeded unexpectedly")
			}
		})
	}
}

func setup(t *testing.T) (repositories.FileRepository, *bytes.Buffer) {
	t.Helper()
	repo, err := sqlite.NewDB("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("failed to init sqlite instance: %v", err)
	}

	buf := new(bytes.Buffer)
	_, err = buf.WriteString("hello, world!")
	if err != nil {
		t.Fatalf("failed to write buffer: %v", err)
	}

	return repo, buf
}
