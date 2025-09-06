package services

import (
	"context"
	"fmt"

	"github.com/portbound/go-fs/internal/models"
	"github.com/portbound/go-fs/internal/repositories"
)

type UserService interface {
	LookupUser(ctx context.Context, id string) (*models.User, error)
}

type userService struct {
	db repositories.UserRepository
}

func NewUserService(userRepo repositories.UserRepository) UserService {
	return &userService{db: userRepo}
}

func (us *userService) LookupUser(ctx context.Context, email string) (*models.User, error) {
	user, err := us.db.GetUser(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("services.GetUser: failed to get user info for email '%s': %w", email, err)
	}
	return user, nil
}
