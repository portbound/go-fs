// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0

package sqlite

import (
	"context"

	"github.com/google/uuid"
)

type Querier interface {
	Create(ctx context.Context, arg CreateParams) error
	Delete(ctx context.Context, id uuid.UUID) error
	Get(ctx context.Context, id uuid.UUID) (File, error)
	GetAll(ctx context.Context) ([]File, error)
}

var _ Querier = (*Queries)(nil)
