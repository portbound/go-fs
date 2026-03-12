package user

import (
	"context"
)

type User struct {
	Id     string `json:"id"`
	Email  string `json:"email"`
	Bucket string `json:"bucket"`
}

type Store interface {
	GetUser(ctx context.Context, email string) (*User, error)
}

