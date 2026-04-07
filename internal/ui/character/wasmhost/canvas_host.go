package wasmhost

import (
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

// CanvasHost abstracts the rendering surface that WASM character plugins draw onto.
type CanvasHost interface {
	SetCircle(id int32, x, y, diameter float64, r, g, b, a uint8)
	RemoveCircle(id int32)
	SetImage(id int32, x, y, w, height float64, pngData []byte)
	RemoveImage(id int32)
	Widget() fyne.CanvasObject
	Refresh()
}

// FyneCanvasHost implements CanvasHost using Fyne canvas objects.
type FyneCanvasHost struct {
	mu        sync.Mutex
	objects   map[int32]fyne.CanvasObject
	container *fyne.Container
}

// NewFyneCanvasHost creates a new FyneCanvasHost with an empty container.
func NewFyneCanvasHost() *FyneCanvasHost {
	return &FyneCanvasHost{
		objects:   make(map[int32]fyne.CanvasObject),
		container: container.NewWithoutLayout(),
	}
}

// SetCircle is a noop stub.
func (h *FyneCanvasHost) SetCircle(id int32, x, y, diameter float64, r, g, b, a uint8) {
}

// RemoveCircle is a noop stub.
func (h *FyneCanvasHost) RemoveCircle(id int32) {
}

// SetImage is a noop stub.
func (h *FyneCanvasHost) SetImage(id int32, x, y, w, height float64, pngData []byte) {
}

// RemoveImage is a noop stub.
func (h *FyneCanvasHost) RemoveImage(id int32) {
}

// Widget returns the container that holds all rendered objects.
func (h *FyneCanvasHost) Widget() fyne.CanvasObject {
	return h.container
}

// Refresh is a noop stub.
func (h *FyneCanvasHost) Refresh() {
}
