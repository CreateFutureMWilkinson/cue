package repository

import (
	"context"

	"github.com/google/uuid"
)

// Category represents a user-defined category for todos.
type Category struct {
	ID    uuid.UUID
	Name  string // unique, user-defined
	Color string // hex color, e.g. "#FF5733"
}

// CategoryRepository defines the contract for category persistence.
type CategoryRepository interface {
	Insert(ctx context.Context, category *Category) error
	Update(ctx context.Context, category *Category) error
	Delete(ctx context.Context, id uuid.UUID) error
	QueryAll(ctx context.Context) ([]*Category, error)
	QueryByName(ctx context.Context, name string) (*Category, error)
}
