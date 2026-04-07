package wasmhost

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"fyne.io/fyne/v2"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// ErrNotImplemented is returned by noop stubs awaiting implementation.
var ErrNotImplemented = errors.New("not implemented")

// WASMCharacterHost implements character.Character by delegating to a WASM plugin
// loaded via wazero. The plugin communicates with the host through the ABI defined
// in abi.go, drawing onto a CanvasHost surface.
type WASMCharacterHost struct {
	mu      sync.Mutex
	name    string
	canvas  CanvasHost
	state   character.CharacterState
	runtime wazero.Runtime
	module  api.Module
	ctx     context.Context
	cancel  context.CancelFunc
}

// LoadPlugin creates a WASMCharacterHost from WASM bytes and a CanvasHost.
// It compiles and instantiates the WASM module, wires up host function imports,
// and reads the plugin name.
func LoadPlugin(wasmBytes []byte, canvas CanvasHost) (*WASMCharacterHost, error) {
	ctx, cancel := context.WithCancel(context.Background())

	rt := wazero.NewRuntime(ctx)

	// Instantiate WASI for basic I/O support.
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, rt); err != nil {
		cancel()
		return nil, fmt.Errorf("instantiating WASI: %w", err)
	}

	host := &WASMCharacterHost{
		canvas:  canvas,
		runtime: rt,
		ctx:     ctx,
		cancel:  cancel,
	}

	// Register host functions the guest can call back into.
	hostMod := rt.NewHostModuleBuilder(HostModuleName)
	hostMod.NewFunctionBuilder().
		WithFunc(func(_ context.Context, id int32, x, y, diameter float64, r, g, b, a int32) {
			canvas.SetCircle(id, x, y, diameter, uint8(r), uint8(g), uint8(b), uint8(a)) // #nosec G115 — WASM ABI color values are always 0-255
		}).Export(ImportHostSetCircle)
	hostMod.NewFunctionBuilder().
		WithFunc(func(_ context.Context, id int32) {
			canvas.RemoveCircle(id)
		}).Export(ImportHostRemoveCircle)
	hostMod.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, id int32, x, y, w, h float64, ptr, length int32) {
			mod := host.module
			if mod == nil {
				return
			}
			data, ok := mod.Memory().Read(uint32(ptr), uint32(length)) // #nosec G115 — WASM i32 values are non-negative by ABI contract
			if !ok {
				return
			}
			canvas.SetImage(id, x, y, w, h, data)
		}).Export(ImportHostSetImage)
	hostMod.NewFunctionBuilder().
		WithFunc(func(_ context.Context, id int32) {
			canvas.RemoveImage(id)
		}).Export(ImportHostRemoveImage)
	hostMod.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, level, ptr, length int32) {
			// host_log: read string from guest memory and log it.
			// For now, this is a no-op; will be wired to activity log later.
		}).Export(ImportHostLog)

	if _, err := hostMod.Instantiate(ctx); err != nil {
		_ = rt.Close(ctx) // #nosec G104 — best-effort cleanup
		cancel()
		return nil, fmt.Errorf("instantiating host module: %w", err)
	}

	// Compile the guest module.
	compiled, err := rt.CompileModule(ctx, wasmBytes)
	if err != nil {
		_ = rt.Close(ctx) // #nosec G104 — best-effort cleanup
		cancel()
		return nil, fmt.Errorf("compiling WASM module: %w", err)
	}

	// Instantiate calling _initialize (not _start) to set up the Go runtime
	// without running main() and exiting. Go 1.24+ exports _initialize when
	// //go:wasmexport functions are present.
	modConfig := wazero.NewModuleConfig().WithStartFunctions("_initialize")
	mod, err := rt.InstantiateModule(ctx, compiled, modConfig)
	if err != nil {
		_ = rt.Close(ctx) // #nosec G104 — best-effort cleanup
		cancel()
		return nil, fmt.Errorf("instantiating WASM module: %w", err)
	}
	host.module = mod

	// Read the plugin name from the guest.
	name, err := host.readPluginName()
	if err != nil {
		_ = rt.Close(ctx) // #nosec G104 — best-effort cleanup
		cancel()
		return nil, fmt.Errorf("reading plugin name: %w", err)
	}
	host.name = name

	return host, nil
}

// readPluginName calls the guest's plugin_name export and reads the string
// from guest memory. The function returns a packed int64: ptr<<32 | len.
func (h *WASMCharacterHost) readPluginName() (string, error) {
	fn := h.module.ExportedFunction(ExportPluginName)
	if fn == nil {
		return "", fmt.Errorf("guest does not export %s", ExportPluginName)
	}

	results, err := fn.Call(h.ctx)
	if err != nil {
		return "", fmt.Errorf("calling %s: %w", ExportPluginName, err)
	}
	if len(results) == 0 {
		return "", fmt.Errorf("%s returned no results", ExportPluginName)
	}

	packed := results[0]
	ptr := uint32(packed >> 32)
	length := uint32(packed & 0xFFFFFFFF)

	if length == 0 {
		return "", nil
	}

	data, ok := h.module.Memory().Read(ptr, length)
	if !ok {
		return "", fmt.Errorf("failed to read plugin name from guest memory at ptr=%d len=%d", ptr, length)
	}

	return string(data), nil
}

// Name returns the character name reported by the WASM plugin.
func (h *WASMCharacterHost) Name() string {
	return h.name
}

// TransitionTo asks the WASM plugin to transition to the given state.
func (h *WASMCharacterHost) TransitionTo(state character.CharacterState) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.state = state

	fn := h.module.ExportedFunction(ExportPluginTransitionTo)
	if fn == nil {
		return
	}
	_, _ = fn.Call(h.ctx, uint64(state)) // #nosec G115 — CharacterState is a small enum (0-5)
}

// CurrentState returns the current state as reported by the WASM plugin.
func (h *WASMCharacterHost) CurrentState() character.CharacterState {
	h.mu.Lock()
	defer h.mu.Unlock()

	fn := h.module.ExportedFunction(ExportPluginCurrentState)
	if fn == nil {
		return h.state
	}
	results, err := fn.Call(h.ctx)
	if err != nil || len(results) == 0 {
		return h.state
	}
	return character.CharacterState(results[0]) // #nosec G115 — CharacterState is a small enum (0-5)
}

// Tick advances the plugin animation by the given number of elapsed milliseconds
// and returns whether the canvas needs a refresh.
func (h *WASMCharacterHost) Tick(elapsedMs int64) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	fn := h.module.ExportedFunction(ExportPluginTick)
	if fn == nil {
		return false
	}
	results, err := fn.Call(h.ctx, uint64(elapsedMs)) // #nosec G115 — elapsed time is always non-negative
	if err != nil || len(results) == 0 {
		return false
	}
	return results[0] != 0
}

// Widget returns the CanvasHost's widget for embedding in the UI.
func (h *WASMCharacterHost) Widget() fyne.CanvasObject {
	return h.canvas.Widget()
}

// Close shuts down the WASM runtime and releases resources.
func (h *WASMCharacterHost) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Call plugin_close in the guest.
	fn := h.module.ExportedFunction(ExportPluginClose)
	if fn != nil {
		_, _ = fn.Call(h.ctx)
	}

	// Shut down the wazero runtime.
	_ = h.runtime.Close(h.ctx)
	h.cancel()
}
