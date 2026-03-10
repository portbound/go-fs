package fs_test

import (
	"context"
	"testing"

	"github.com/portbound/go-fs/internal/fs"
)

func TestService_Upload(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		meta  fs.MetaStore
		media fs.MediaStore
		// Named input parameters for target function.
		requests <-chan fs.UploadRequest
		want     <-chan fs.UploadResult
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := fs.NewService(tt.meta, tt.media)
			got := s.Upload(context.Background(), tt.requests)
			// TODO: update the condition below to compare got with tt.want.
			if true {
				t.Errorf("Upload() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_Download(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		meta  fs.MetaStore
		media fs.MediaStore
		// Named input parameters for target function.
		request fs.DownloadRequest
		want    *fs.DownloadResult
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := fs.NewService(tt.meta, tt.media)
			got, gotErr := s.Download(context.Background(), tt.request)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Download() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("Download() succeeded unexpectedly")
			}
			// TODO: update the condition below to compare got with tt.want.
			if true {
				t.Errorf("Download() = %v, want %v", got, tt.want)
			}
		})
	}
}
