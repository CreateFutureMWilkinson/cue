package uitest

import (
	"testing"

	"fyne.io/fyne/v2"
)

// FindWidget walks a fyne.CanvasObject tree depth-first and returns the first
// widget matching type T and predicate.
func FindWidget[T fyne.CanvasObject](root fyne.CanvasObject, predicate func(T) bool) (T, bool) {
	var zero T
	if root == nil {
		return zero, false
	}

	if typed, ok := root.(T); ok && predicate(typed) {
		return typed, true
	}

	for _, child := range children(root) {
		if found, ok := FindWidget[T](child, predicate); ok {
			return found, true
		}
	}

	return zero, false
}

// RequireWidget calls FindWidget and fails the test if not found.
func RequireWidget[T fyne.CanvasObject](t *testing.T, root fyne.CanvasObject, predicate func(T) bool) T {
	t.Helper()
	found, ok := FindWidget[T](root, predicate)
	if !ok {
		t.Fatalf("RequireWidget: no matching widget of type %T found in tree", *new(T))
	}
	return found
}

// FindAll returns all widgets matching type T and predicate.
func FindAll[T fyne.CanvasObject](root fyne.CanvasObject, predicate func(T) bool) []T {
	var results []T
	findAll[T](root, predicate, &results)
	return results
}

func findAll[T fyne.CanvasObject](root fyne.CanvasObject, predicate func(T) bool, results *[]T) {
	if root == nil {
		return
	}

	if typed, ok := root.(T); ok && predicate(typed) {
		*results = append(*results, typed)
	}

	for _, child := range children(root) {
		findAll[T](child, predicate, results)
	}
}

// children returns the child objects of a canvas object. It handles
// *fyne.Container directly. For other types that hold children, it checks
// for the common Objects() pattern.
func children(obj fyne.CanvasObject) []fyne.CanvasObject {
	if obj == nil {
		return nil
	}

	if c, ok := obj.(*fyne.Container); ok {
		return c.Objects
	}

	// Support any widget or object that exposes an Objects() method.
	type hasObjects interface {
		Objects() []fyne.CanvasObject
	}
	if provider, ok := obj.(hasObjects); ok {
		return provider.Objects()
	}

	return nil
}
