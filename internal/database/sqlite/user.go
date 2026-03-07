package sqlite

import (
	"context"

	"github.com/portbound/go-fs/internal/user"
)

func (db *SQLiteDB) GetUser(ctx context.Context, email string) (*user.User, error) {
	data, err := db.Queries.GetUser(ctx, email)
	if err != nil {
		return nil, err
	}

	return &user.User{
		ID:         data.ID,
		Email:      data.Email,
		BucketName: data.BucketName,
	}, nil
}
