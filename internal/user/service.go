package user

import (
	"context"
	"time"
)

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
