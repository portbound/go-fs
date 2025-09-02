package repositories

import (
	"context"

	"github.com/portbound/go-fs/internal/models"
)

type UserRepository interface {
	GetUser(ctx context.Context, token string) (*models.User, error)
}
