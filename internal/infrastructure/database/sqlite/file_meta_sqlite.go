// Package sqlite
package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/portbound/go-fs/internal/models"
)

func (db *SQLiteDB) CreateFileMeta(ctx context.Context, filemeta *models.FileMeta) error {
	params := CreateFileMetaParams{
		ID:          filemeta.ID,
		ParentID:    sql.NullString{String: filemeta.ParentID, Valid: true},
		ThumbID:     sql.NullString{String: filemeta.ThumbID, Valid: true},
		Name:        filemeta.Name,
		ContentType: filemeta.ContentType,
		Size:        filemeta.Size,
		UploadDate:  filemeta.UploadDate.Format(time.RFC3339),
		Owner:       filemeta.Owner,
	}
	return db.Queries.CreateFileMeta(ctx, params)
}

func (db *SQLiteDB) GetFileMeta(ctx context.Context, id string) (*models.FileMeta, error) {
	file, err := db.Queries.GetFileMeta(ctx, id)
	if err != nil {
		return nil, err
	}
	return mapFileMeta(file)
}

func (db *SQLiteDB) GetAllFileMeta(ctx context.Context, owner *models.User) ([]*models.FileMeta, error) {
	data, err := db.Queries.GetAllFileMeta(ctx, owner.Email)
	if err != nil {
		return nil, err
	}

	var files []*models.FileMeta
	for _, row := range data {
		file, err := mapFileMeta(row)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

func (db *SQLiteDB) DeleteFileMeta(ctx context.Context, id string) error {
	return db.Queries.DeleteFileMeta(ctx, id)
}

func mapFileMeta(f FileMetum) (*models.FileMeta, error) {
	uploadDate, _ := time.Parse(time.RFC3339, f.UploadDate)
	return &models.FileMeta{
		ID:          f.ID,
		ParentID:    f.ParentID.String,
		ThumbID:     f.ThumbID.String,
		Name:        f.Name,
		ContentType: f.ContentType,
		Size:        f.Size,
		UploadDate:  uploadDate,
		Owner:       f.Owner,
	}, nil
}
