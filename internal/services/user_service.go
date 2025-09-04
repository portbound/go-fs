package services

import (
	"context"

	"github.com/portbound/go-fs/internal/models"
	"github.com/portbound/go-fs/internal/repositories"
)

type UserService struct {
	db repositories.UserRepository
}

func NewUserService(userRepo repositories.UserRepository) *UserService {
	return &UserService{db: userRepo}
}

func (us *UserService) GetUser(ctx context.Context, email string) (*models.User, error) {
	return us.db.GetUser(ctx, email)
}
