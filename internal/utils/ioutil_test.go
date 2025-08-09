package utils_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/portbound/go-fs/internal/utils"
)

func TestStageFileToDisk(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		fileName string
		reader   io.Reader
		want     string
		wantErr  bool
	}{
		{
			name:     "Happy Path",
			path:     t.TempDir(),
			fileName: "test.txt",
			reader:   strings.NewReader("hello world"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := utils.StageFileToDisk(context.Background(), tt.path, tt.fileName, tt.reader)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("StageFileToDisk() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("StageFileToDisk() succeeded unexpectedly")
			}

			want := filepath.Join(tt.path, tt.fileName)
			if got != want {
				t.Errorf("StageFileToDisk() = %v, want %v", got, want)
			}

			data, err := os.ReadFile(got)
			if err != nil {
				t.Fatalf("failed to read created file: %v", err)
			}
			if string(data) != "hello world" {
				t.Errorf("file content = %q, want %q", string(data), "hello world")
			}
		})
	}
}
