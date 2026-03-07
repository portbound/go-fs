package user

import (
	"context"
	"time"
)

type User struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	BucketName string `json:"bucketName"`
}

type Store interface {
	GetUser(ctx context.Context, email string) (*User, error)
}

type Service struct {
	store Store
}

func NewService(s Store) *Service {
	return &Service{store: s}
}

func (s *Service) ByEmail(ctx context.Context, email string) (*User, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	user, err := s.store.GetUser(dbCtx, email)
	if err != nil {
		return nil, err
	}
	return user, nil
}
