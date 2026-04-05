package uitest

import (
	"errors"
	"testing"

	"fyne.io/fyne/v2"
)

var ErrNotImplemented = errors.New("not implemented")

// FindWidget walks a fyne.CanvasObject tree and returns the first widget
// matching type T and predicate.
func FindWidget[T fyne.CanvasObject](root fyne.CanvasObject, predicate func(T) bool) (T, bool) {
	var zero T
	return zero, false
}

// RequireWidget calls FindWidget and fails the test if not found.
func RequireWidget[T fyne.CanvasObject](t *testing.T, root fyne.CanvasObject, predicate func(T) bool) T {
	var zero T
	return zero
}

// FindAll returns all widgets matching type T and predicate.
func FindAll[T fyne.CanvasObject](root fyne.CanvasObject, predicate func(T) bool) []T {
	return nil
}
