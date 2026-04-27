package repository

import (
	"context"
	"time"
)

// Category represents a user-defined category for tasks.
//
// Categories are identified by their normalized lowercase NameKey
// (see NormalizeCategoryKey) rather than by UUID. The presentation
// form is derived from the key via PresentCategoryName.
type Category struct {
	NameKey   string  // primary key; lowercase, '_' for spaces
	Colour    *string // nullable hex '#RRGGBB'
	CreatedAt time.Time
}

// CategoryWithCount is a Category enriched with the number of tasks
// referencing it.
type CategoryWithCount struct {
	Category
	TaskCount int
}

// CategoryRepository defines the contract for category persistence.
//
// All methods take canonical (already-normalized) keys; raw user input
// must be passed through NormalizeCategoryKey at the service boundary
// before reaching the repository.
type CategoryRepository interface {
	Insert(ctx context.Context, c *Category) error
	Rename(ctx context.Context, oldKey, newKey string) error
	UpdateColour(ctx context.Context, key string, colour *string) error
	Delete(ctx context.Context, key string) error
	GetByKey(ctx context.Context, key string) (*Category, error)
	QueryAll(ctx context.Context, withCounts bool) ([]*CategoryWithCount, error)
}

// NormalizeCategoryKey converts raw user input to the canonical
// lowercase, underscore-separated form used as the category PK.
//
// Rules (per Feature 109 Decision 3):
//  1. Trim leading/trailing whitespace.
//  2. Reject if input contains '_' (error includes "underscores not allowed").
//  3. Reject if empty after trim, longer than 64 chars, or contains
//     anything other than ASCII letters, digits, or whitespace.
//  4. Lowercase.
//  5. Collapse runs of whitespace, replace each run with a single '_'.
//
// Stub: returns ErrNotImplemented; replaced in Loop 1 GREEN.
func NormalizeCategoryKey(input string) (string, error) {
	return "", ErrNotImplemented
}

// PresentCategoryName converts a normalized category key back to a
// human-readable display string by replacing '_' with space and
// title-casing each word. Empty input returns empty string.
//
// Stub: returns empty string; replaced in Loop 1 GREEN.
func PresentCategoryName(key string) string {
	return ""
}
