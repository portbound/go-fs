package sqlite

import (
	"context"

	"github.com/portbound/go-fs/internal/models"
)

func (db *SQLiteDB) GetUser(ctx context.Context, email string) (*models.User, error) {
	data, err := db.Queries.GetUser(ctx, email)
	if err != nil {
		return nil, err
	}
	return mapToUser(data), nil
}

func mapToUser(u User) *models.User {
	return &models.User{
		ID:         u.ID,
		Email:      u.Email,
		Token:      u.Token.String,
		BucketName: u.BucketName,
	}

}
