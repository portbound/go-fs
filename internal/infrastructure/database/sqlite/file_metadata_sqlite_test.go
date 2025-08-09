package sqlite_test

import (
	"context"
	"database/sql"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/portbound/go-fs/internal/infrastructure/database/sqlite"
	"github.com/portbound/go-fs/internal/models"
)

func TestFileMetadataRepository(t *testing.T) {
	db, err := sqlite.NewDB(":memory:")
	if err != nil {
		t.Fatalf("could not construct receiver type: %v", err)
	}

	fm := &models.FileMeta{
		ID:          uuid.New(),
		Name:        "test.txt",
		Owner:       "test-owner",
		ContentType: "text/plain",
		FilePath:    "/test.txt",
		ThumbPath:   "/thumb.jpg",
	}

	t.Run("Create", func(t *testing.T) {
		err := db.Create(context.Background(), fm)
		if err != nil {
			t.Fatalf("Create() failed: %v", err)
		}

		got, err := db.Get(context.Background(), fm.ID)
		if err != nil {
			t.Fatalf("Get() after Create() failed: %v", err)
		}

		// these fields are not part of the DB schema for file metadata so we set them to their zero value before comparison
		fm.TmpFilePath = ""
		fm.TmpThumbPath = ""
		fm.PreviewPath = ""

		if !reflect.DeepEqual(got, fm) {
			t.Errorf("Get() after Create() = %v, want %v", got, fm)
		}
	})

	t.Run("Get", func(t *testing.T) {
		got, err := db.Get(context.Background(), fm.ID)
		if err != nil {
			t.Fatalf("Get() failed: %v", err)
		}

		if !reflect.DeepEqual(got, fm) {
			t.Errorf("Get() = %v, want %v", got, fm)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		err := db.Delete(context.Background(), fm.ID)
		if err != nil {
			t.Fatalf("Delete() failed: %v", err)
		}

		_, err = db.Get(context.Background(), fm.ID)
		if err == nil {
			t.Fatal("Get() after Delete() succeeded unexpectedly")
		}
		if err != sql.ErrNoRows {
			t.Fatalf("expected sql.ErrNoRows but got: %v", err)
		}
	})
}
