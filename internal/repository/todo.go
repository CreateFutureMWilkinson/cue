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
	Priority           int        // higher = higher priority
	DueDate            *time.Time // optional
	Categories         []Category
	EstimateMinutes    *int // user-provided time estimate
	LLMEstimateMinutes *int // LLM-generated time estimate
	CreatedAt          time.Time
	CompletedAt        *time.Time // nil = incomplete
}

// TodoFilter controls filtering, searching, and pagination for QueryFiltered.
type TodoFilter struct {
	Status   string // "incomplete", "complete", "all" (default: "incomplete")
	Category string // filter by category name (empty = no filter)
	Search   string // LIKE match against title AND description (empty = no filter)
	Limit    int    // page size (default: 50)
	Offset   int    // pagination offset (default: 0)
}

// TodoRepository defines the contract for todo persistence.
type TodoRepository interface {
	Insert(ctx context.Context, todo *Todo) error
	Update(ctx context.Context, todo *Todo) error
	Delete(ctx context.Context, id uuid.UUID) error
	QueryByID(ctx context.Context, id uuid.UUID) (*Todo, error)
	QueryFiltered(ctx context.Context, filter TodoFilter) ([]*Todo, int, error)
	Complete(ctx context.Context, id uuid.UUID, completedAt time.Time) error
}
