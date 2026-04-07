package wasmhost

import (
	"bytes"
	"image/color"
	"slices"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	fynecontainer "fyne.io/fyne/v2/container"
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
		container: fynecontainer.NewWithoutLayout(),
	}
}

// rebuildContainer replaces the container's Objects slice with the current
// objects map entries sorted by ID for deterministic z-ordering.
func (h *FyneCanvasHost) rebuildContainer() {
	ids := make([]int32, 0, len(h.objects))
	for id := range h.objects {
		ids = append(ids, id)
	}
	slices.Sort(ids)

	objs := make([]fyne.CanvasObject, 0, len(ids))
	for _, id := range ids {
		objs = append(objs, h.objects[id])
	}
	h.container.Objects = objs
}

// SetCircle creates or updates a circle with the given ID, position, size, and color.
func (h *FyneCanvasHost) SetCircle(id int32, x, y, diameter float64, r, g, b, a uint8) {
	h.mu.Lock()
	defer h.mu.Unlock()

	fillColor := color.RGBA{R: r, G: g, B: b, A: a}

	if existing, ok := h.objects[id]; ok {
		if circle, isCircle := existing.(*canvas.Circle); isCircle {
			circle.FillColor = fillColor
			circle.Move(fyne.NewPos(float32(x), float32(y)))
			circle.Resize(fyne.NewSize(float32(diameter), float32(diameter)))
			return
		}
	}

	circle := canvas.NewCircle(fillColor)
	circle.Move(fyne.NewPos(float32(x), float32(y)))
	circle.Resize(fyne.NewSize(float32(diameter), float32(diameter)))

	h.objects[id] = circle
	h.rebuildContainer()
}

// RemoveCircle removes the circle with the given ID from the container.
func (h *FyneCanvasHost) RemoveCircle(id int32) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.objects, id)
	h.rebuildContainer()
}

// SetImage creates or updates an image with the given ID, position, size, and PNG data.
func (h *FyneCanvasHost) SetImage(id int32, x, y, w, height float64, pngData []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	img := canvas.NewImageFromReader(bytes.NewReader(pngData), "img")
	img.FillMode = canvas.ImageFillOriginal
	img.Move(fyne.NewPos(float32(x), float32(y)))
	img.Resize(fyne.NewSize(float32(w), float32(height)))

	h.objects[id] = img
	h.rebuildContainer()
}

// RemoveImage removes the image with the given ID from the container.
func (h *FyneCanvasHost) RemoveImage(id int32) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.objects, id)
	h.rebuildContainer()
}

// Widget returns the container that holds all rendered objects.
func (h *FyneCanvasHost) Widget() fyne.CanvasObject {
	return h.container
}

// Refresh triggers a re-render of the container and all its children.
func (h *FyneCanvasHost) Refresh() {
	h.container.Refresh()
}
