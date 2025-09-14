package services

import (
	"context"
	"time"

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
	dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	user, err := us.db.GetUser(dbCtx, email)
	if err != nil {
		return nil, err
	}
	return user, nil
}
