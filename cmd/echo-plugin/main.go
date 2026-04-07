//go:build wasip1

package main

import "unsafe"

// Host function imports from the "cue" module.

//go:wasmimport cue host_set_circle
func hostSetCircle(id int32, x float64, y float64, diameter float64, r int32, g int32, b int32, a int32)

//go:wasmimport cue host_remove_circle
func hostRemoveCircle(id int32)

// Global state.
var (
	pluginNameStr = "echo"
	currentState  int32
)

//go:wasmexport plugin_name
func pluginName() int64 {
	ptr := int64(uintptr(unsafe.Pointer(unsafe.StringData(pluginNameStr))))
	return ptr<<32 | int64(len(pluginNameStr))
}

//go:wasmexport plugin_init
func pluginInit(width float64, height float64) {
}

//go:wasmexport plugin_transition_to
func pluginTransitionTo(state int32) {
	currentState = state
}

//go:wasmexport plugin_current_state
func pluginCurrentState() int32 {
	return currentState
}

//go:wasmexport plugin_tick
func pluginTick(elapsedMs int64) int32 {
	hostSetCircle(0, 0.5, 0.5, 0.1, 0, 255, 0, 255)
	return 1
}

//go:wasmexport plugin_close
func pluginClose() {
	hostRemoveCircle(0)
}

//go:wasmexport cue_allocate
func cueAllocate(size int32) int32 {
	buf := make([]byte, size)
	return int32(uintptr(unsafe.Pointer(&buf[0])))
}

func main() {}
