package sqlite_test

import (
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/portbound/go-fs/internal/infrastructure/database/sqlite"
	"github.com/portbound/go-fs/internal/models"
)

func TestDB_Create(t *testing.T) {
	db, err := sqlite.NewDB("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("could not construct receiver type: %v", err)
	}
	tests := []struct {
		name     string
		filemeta *models.FileMeta
		wantErr  bool
	}{
		{
			name:     "happy-path",
			filemeta: seedFilemeta(t),
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			t.Cleanup(func() {

			})

			gotErr := db.Create(t.Context(), tt.filemeta)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Create() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("Create() succeeded unexpectedly")
			}
		})
	}
}

func TestDB_Get(t *testing.T) {
	db, err := sqlite.NewDB("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("could not construct receiver type: %v", err)
	}

	tests := []struct {
		name    string
		connStr string
		want    *models.FileMeta
		wantErr bool
	}{
		{
			name:    "happy-path",
			connStr: "file::memory:?cache=shared",
			want:    seedFilemeta(t),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := db.Create(t.Context(), tt.want); err != nil {
				t.Errorf("Get() failed to seed db: %v", err)
			}

			got, gotErr := db.Get(t.Context(), tt.want.ID)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Get() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("Get() succeeded unexpectedly")
			}
			// TODO: update the condition below to compare got with tt.want.
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func seedFilemeta(t *testing.T) *models.FileMeta {
	t.Helper()
	return &models.FileMeta{
		ID:           uuid.New(),
		Name:         "test.txt",
		Owner:        "tester",
		ContentType:  "plain/text",
		Size:         10,
		OriginalPath: "test-path",
		TmpDir:       "test-dir",
	}
}

func resetTables(t *testing.T) error {

	return nil
}
