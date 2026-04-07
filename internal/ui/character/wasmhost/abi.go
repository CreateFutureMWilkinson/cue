package wasmhost

// ABI constants for the WASM character plugin interface.
const (
	// HostModuleName is the WASM import module name for host functions.
	HostModuleName = "cue"

	// Guest exports (host calls into plugin)
	ExportPluginName         = "plugin_name"
	ExportPluginInit         = "plugin_init"
	ExportPluginTransitionTo = "plugin_transition_to"
	ExportPluginCurrentState = "plugin_current_state"
	ExportPluginTick         = "plugin_tick"
	ExportPluginClose        = "plugin_close"
	ExportCueAllocate        = "cue_allocate"

	// Host imports (plugin calls into host)
	ImportHostSetCircle    = "host_set_circle"
	ImportHostRemoveCircle = "host_remove_circle"
	ImportHostSetImage     = "host_set_image"
	ImportHostRemoveImage  = "host_remove_image"
	ImportHostLog          = "host_log"
)
