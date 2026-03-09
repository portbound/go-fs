package sqlite

import (
	"context"
	"database/sql"

	"github.com/portbound/go-fs/internal/fs"
)

func (db *SQLiteDB) Save(ctx context.Context, m *fs.Metadata) error {
	params := SaveMetadataParams{
		ID:        m.Id,
		FileName:  m.Filename,
		ThumbName: sql.NullString{String: m.Thumbname, Valid: true},
		UserID:    m.UserId,
	}

	return db.Queries.SaveMetadata(ctx, params)
}

func (db *SQLiteDB) Get(ctx context.Context, id, userId string) (*fs.Metadata, error) {
	params := GetMetadataParams{
		ID:     id,
		UserID: userId,
	}

	f, err := db.Queries.GetMetadata(ctx, params)
	if err != nil {
		return nil, err
	}

	return &fs.Metadata{
		Id:        f.ID,
		Filename:  f.FileName,
		Thumbname: f.ThumbName.String,
		UserId:    f.UserID,
	}, nil
}

func (db *SQLiteDB) Update(ctx context.Context, id, email string)

func (db *SQLiteDB) Delete(ctx context.Context, id, email string) error {
	params := DeleteMetadataParams{
		ID:     id,
		UserID: email,
	}

	return db.Queries.DeleteMetadata(ctx, params)
}
