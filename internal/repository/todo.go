package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Todo represents a task in the todo list.
type Todo struct {
	ID                 uuid.UUID
	Title              string
	Description        string     // markdown
	Priority           int        // lower = higher priority
	DueDate            *time.Time // optional
	Categories         []Category
	EstimateMinutes    *int // user-provided time estimate
	LLMEstimateMinutes *int // LLM-generated time estimate
	CreatedAt          time.Time
	CompletedAt        *time.Time // nil = incomplete
}

// TodoRepository defines the contract for todo persistence.
type TodoRepository interface {
	Insert(ctx context.Context, todo *Todo) error
	Update(ctx context.Context, todo *Todo) error
	Delete(ctx context.Context, id uuid.UUID) error
	QueryByID(ctx context.Context, id uuid.UUID) (*Todo, error)
	QueryIncomplete(ctx context.Context) ([]*Todo, error)
	QueryAll(ctx context.Context) ([]*Todo, error)
	Complete(ctx context.Context, id uuid.UUID, completedAt time.Time) error
}
