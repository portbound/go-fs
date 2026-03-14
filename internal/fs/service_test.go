package fs_test

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/portbound/go-fs/internal/fs"
)

func TestService_Upload(t *testing.T) {
	tests := []struct {
		name    string
		meta    fs.MetaStore
		media   fs.MediaStore
		files   []string
		want    <-chan fs.UploadResult
		wantErr bool
	}{
		{
			name:    "single image",
			meta:    fs.NewMockMetaStore(),
			media:   fs.NewMockMediaStore(),
			files:   []string{"yellow-circle.jpg"},
			want:    make(<-chan fs.UploadResult),
			wantErr: false,
		},
		{
			name:    "single video",
			meta:    fs.NewMockMetaStore(),
			media:   fs.NewMockMediaStore(),
			files:   []string{"3013010-uhd_3840_2160_25fps.mp4"},
			want:    make(<-chan fs.UploadResult),
			wantErr: false,
		},
		{
			name:    "many images and video",
			meta:    fs.NewMockMetaStore(),
			media:   fs.NewMockMediaStore(),
			files:   []string{"yellow-circle.jpg", "thruster.png", "3013010-uhd_3840_2160_25fps.mp4", "1766260_otrebot_drawing-of-ness.png"},
			want:    make(<-chan fs.UploadResult),
			wantErr: false,
		},
		{
			name:    "invalid file type",
			meta:    fs.NewMockMetaStore(),
			media:   fs.NewMockMediaStore(),
			files:   []string{"invalid-file.txt"},
			want:    make(<-chan fs.UploadResult),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := fs.NewService(tt.meta, tt.media)
			requests := make(chan fs.UploadRequest)
			results := s.Upload(context.Background(), requests)
			var files []*os.File

			defer func() {
				for _, f := range files {
					f.Close()
				}
			}()

			go func() {
				defer close(requests)
				buf := make([]byte, 512)
				for _, filename := range tt.files {
					path := filepath.Join("testdata", filename)
					f, err := os.Open(path)
					if err != nil {
					}
					files = append(files, f)

					n, err := f.Read(buf)
					if err != nil {
					}

					contentType := http.DetectContentType(buf[:n])

					_, err = f.Seek(0, 0)
					if err != nil {
					}

					requests <- fs.UploadRequest{
						Reader:      f,
						Filename:    filepath.Base(f.Name()),
						ContentType: contentType,
						UserId:      "test_user",
						Bucket:      "test_bucket",
					}
				}
			}()

			for result := range results {
				if result.Err != nil {
					if !tt.wantErr {
						t.Errorf("unexpected error: %v", result.Err)
					}
				}
			}
		})
	}
}
