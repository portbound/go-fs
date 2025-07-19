package services_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/portbound/go-fs/internal/infrastructure/database/sqlite"
	"github.com/portbound/go-fs/internal/models"
	"github.com/portbound/go-fs/internal/repositories"
	"github.com/portbound/go-fs/internal/services"
	"golang.org/x/net/context"
)

func TestFileService_StageFileToDisk(t *testing.T) {
	tests := []struct {
		name             string
		repo             repositories.FileRepository
		localStoragePath string
		ctx              context.Context
		metadata         *models.FileMeta
		mp               io.Reader
		wantErr          bool
	}{
		{
			name:             "passing",
			localStoragePath: t.TempDir(),
			ctx:              t.Context(),
			metadata: &models.FileMeta{
				Name:  "test",
				Owner: "test",
				Type:  "text/plain",
			},
			mp:      nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := services.NewFileService(tt.repo, tt.localStoragePath)
			repo, buf := setup(t)
			tt.repo = repo
			tt.mp = buf
			gotErr := fs.StageFileToDisk(tt.ctx, tt.metadata, tt.mp)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("StageFileToDisk() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("StageFileToDisk() succeeded unexpectedly")
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
		t.Fatalf("failed to write to buffer: %v", err)
	}

	return repo, buf
}
