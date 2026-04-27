package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
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
	s := strings.TrimSpace(input)
	if strings.ContainsRune(s, '_') {
		return "", errors.New("underscores not allowed in category name")
	}
	if s == "" {
		return "", errors.New("category name must not be empty")
	}
	if len(s) > 64 {
		return "", fmt.Errorf("category name exceeds 64 characters: %d", len(s))
	}
	for _, r := range s {
		if r > unicode.MaxASCII {
			return "", fmt.Errorf("category name contains non-ASCII character: %q", r)
		}
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r)) {
			return "", fmt.Errorf("category name contains invalid character: %q", r)
		}
	}
	s = strings.ToLower(s)
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteRune('_')
			}
			prevSpace = true
			continue
		}
		b.WriteRune(r)
		prevSpace = false
	}
	return b.String(), nil
}

// PresentCategoryName converts a normalized category key back to a
// human-readable display string by replacing '_' with space and
// title-casing each word. Empty input returns empty string.
//
// Stub: returns empty string; replaced in Loop 1 GREEN.
func PresentCategoryName(key string) string {
	if key == "" {
		return ""
	}
	parts := strings.Split(key, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		runes := []rune(p)
		runes[0] = unicode.ToUpper(runes[0])
		for j := 1; j < len(runes); j++ {
			runes[j] = unicode.ToLower(runes[j])
		}
		parts[i] = string(runes)
	}
	return strings.Join(parts, " ")
}
