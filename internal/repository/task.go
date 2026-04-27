package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Task represents a task in the todo list.
type Task struct {
	ID                 uuid.UUID
	Title              string
	Description        string // markdown
	Priority           int    // higher = higher priority
	DueDate            *time.Time
	CategoryKey        *string // FK to categories.name_key; nullable
	EstimateMinutes    *int    // user-provided time estimate
	LLMEstimateMinutes *int    // LLM-generated time estimate
	CreatedAt          time.Time
	CompletedAt        *time.Time // nil = incomplete
}

// TaskFilter controls filtering, searching, and pagination for QueryFiltered.
type TaskFilter struct {
	Status      string // "incomplete", "complete", "all" (default: "incomplete")
	CategoryKey string // canonical category key (empty = no filter)
	Search      string // LIKE match against title AND description (empty = no filter)
	Limit       int    // page size (default: 50)
	Offset      int    // pagination offset (default: 0)
}

// TaskRepository defines the contract for task persistence.
type TaskRepository interface {
	Insert(ctx context.Context, task *Task) error
	Update(ctx context.Context, task *Task) error
	Delete(ctx context.Context, id uuid.UUID) error
	QueryByID(ctx context.Context, id uuid.UUID) (*Task, error)
	QueryFiltered(ctx context.Context, filter TaskFilter) ([]*Task, int, error)
	Complete(ctx context.Context, id uuid.UUID, completedAt time.Time) error
}
