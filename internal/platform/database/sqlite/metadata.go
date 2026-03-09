package sqlite

import (
	"context"

	"github.com/portbound/go-fs/internal/fs"
)

func (db *SQLiteDB) Save(ctx context.Context, m *fs.Metadata) error {
	params := SaveMetadataParams{
		ID:        m.Id,
		FileName:  m.Filename,
		ThumbName: m.Thumbname,
		UserID:    m.UserId,
	}

	return db.Queries.SaveMetadata(ctx, params)
}

func (db *SQLiteDB) Get(ctx context.Context, id, userId string) (*fs.Metadata, error) {
	params := GetMetadataParams{
		ID:     id,
		UserID: userId,
	}

	m, err := db.Queries.GetMetadata(ctx, params)
	if err != nil {
		return nil, err
	}

	return &fs.Metadata{
		Id:        m.ID,
		Filename:  m.FileName,
		Thumbname: m.ThumbName,
		UserId:    m.UserID,
	}, nil
}

func (db *SQLiteDB) GetAll(ctx context.Context, userId string) ([]*fs.Metadata, error) {
	rows, err := db.GetAllMetadata(ctx, userId)
	if err != nil {
		return nil, err
	}

	results := make([]*fs.Metadata, len(rows))
	for i, m := range rows {
		results[i] = &fs.Metadata{
			Id:        m.ID,
			Filename:  m.FileName,
			Thumbname: m.ThumbName,
			UserId:    m.UserID,
		}
	}

	return results, nil
}

func (db *SQLiteDB) Delete(ctx context.Context, id, email string) error {
	params := DeleteMetadataParams{
		ID:     id,
		UserID: email,
	}

	return db.Queries.DeleteMetadata(ctx, params)
}
