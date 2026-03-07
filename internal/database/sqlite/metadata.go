// Package sqlite
package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/portbound/go-fs/internal/fs"
	"github.com/portbound/go-fs/internal/user"
)

func (db *SQLiteDB) Save(ctx context.Context, m *fs.Metadata) error {
	params := CreateFileMetaParams{
		ID:          m.ID,
		ParentID:    sql.NullString{String: m.ParentID, Valid: true},
		ThumbID:     sql.NullString{String: m.ThumbID, Valid: true},
		Name:        m.Name,
		ContentType: m.ContentType,
		Size:        m.Size,
		UploadDate:  m.UploadDate.Format(time.RFC3339),
		Owner:       m.Owner,
	}

	return db.Queries.CreateFileMeta(ctx, params)
}

func (db *SQLiteDB) Get(ctx context.Context, id string, owner *user.User) (*fs.Metadata, error) {
	params := GetFileMetaParams{
		ID:    id,
		Owner: owner.Email,
	}

	file, err := db.Queries.GetFileMeta(ctx, params)
	if err != nil {
		return nil, err
	}

	return mapFileMeta(file)
}

// func (db *SQLiteDB) GetByNameAndOwner(ctx context.Context, name string, owner *user.User) (*fs.Metadata, error) {
// 	params := GetFileMetaByNameAndOwnerParams{
// 		Name:  name,
// 		Owner: owner.Email,
// 	}
//
// 	file, err := db.Queries.GetFileMetaByNameAndOwner(ctx, params)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	return mapFileMeta(file)
// }

// func (db *SQLiteDB) GetAll(ctx context.Context, owner *user.User) ([]*fs.Metadata, error) {
// 	data, err := db.Queries.GetAllFileMeta(ctx, owner.Email)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	var files []*fs.Metadata
// 	for _, row := range data {
// 		file, err := mapFileMeta(row)
// 		if err != nil {
// 			return nil, err
// 		}
// 		files = append(files, file)
// 	}
// 	return files, nil
// }

func (db *SQLiteDB) Delete(ctx context.Context, id string, owner *user.User) error {
	params := DeleteFileMetaParams{
		ID:    id,
		Owner: owner.Email,
	}

	return db.Queries.DeleteFileMeta(ctx, params)
}

func mapFileMeta(f FileMetum) (*fs.Metadata, error) {
	uploadDate, err := time.Parse(time.RFC3339, f.UploadDate)
	if err != nil {
		return nil, err
	}

	return &fs.Metadata{
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
