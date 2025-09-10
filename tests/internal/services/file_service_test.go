package services_test

import (
	"context"
	"testing"

	"github.com/portbound/go-fs/internal/models"
	"github.com/portbound/go-fs/internal/services"
)

func Test_fileService_ProcessBatch(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		batch []*models.FileMeta
		owner *models.User
		want  []error
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: construct the receiver type with mock dependencies.
			var fs services.FileService
			got := fs.ProcessBatch(context.Background(), tt.batch, tt.owner)
			// TODO: update the condition below to compare got with tt.want.
			if got != nil {
				t.Errorf("ProcessBatch() = %v, want %v", got, tt.want)
			}
		})
	}
}
