package wasmhost

import (
	"errors"

	"fyne.io/fyne/v2"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
)

// ErrNotImplemented is returned by noop stubs awaiting implementation.
var ErrNotImplemented = errors.New("not implemented")

// WASMCharacterHost implements character.Character by delegating to a WASM plugin
// loaded via wazero. The plugin communicates with the host through the ABI defined
// in abi.go, drawing onto a CanvasHost surface.
type WASMCharacterHost struct {
	name   string
	canvas CanvasHost
	state  character.CharacterState
}

// LoadPlugin creates a WASMCharacterHost from WASM bytes and a CanvasHost.
// It compiles and instantiates the WASM module, wires up host function imports,
// and calls plugin_init.
func LoadPlugin(wasmBytes []byte, canvas CanvasHost) (*WASMCharacterHost, error) {
	return nil, ErrNotImplemented
}

// Name returns the character name reported by the WASM plugin.
func (h *WASMCharacterHost) Name() string {
	return ""
}

// TransitionTo asks the WASM plugin to transition to the given state.
func (h *WASMCharacterHost) TransitionTo(state character.CharacterState) {
}

// CurrentState returns the current state as reported by the WASM plugin.
func (h *WASMCharacterHost) CurrentState() character.CharacterState {
	return character.StateIdle
}

// Tick advances the plugin animation by the given number of elapsed milliseconds
// and returns whether the canvas needs a refresh.
func (h *WASMCharacterHost) Tick(elapsedMs int64) bool {
	return false
}

// Widget returns the CanvasHost's widget for embedding in the UI.
func (h *WASMCharacterHost) Widget() fyne.CanvasObject {
	return nil
}

// Close shuts down the WASM runtime and releases resources.
func (h *WASMCharacterHost) Close() {
}
