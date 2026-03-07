package user

import (
	"context"
	"time"
)

type User struct {
	Id     string `json:"id"`
	Email  string `json:"email"`
	Bucket string `json:"bucket"`
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

func (s *Service) Get(ctx context.Context, email string) (*User, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	user, err := s.store.GetUser(dbCtx, email)
	if err != nil {
		return nil, err
	}
	return user, nil
}
