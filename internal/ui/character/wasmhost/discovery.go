package wasmhost

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
)

// PluginInfo describes a discovered WASM character plugin.
type PluginInfo struct {
	Name string // filename without .wasm extension
	Path string // full path to .wasm file
}

// DiscoverPlugins scans a directory for .wasm files and returns info for each.
// Returns an empty slice (no error) if the directory doesn't exist.
func DiscoverPlugins(dir string) ([]PluginInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var plugins []PluginInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".wasm" {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".wasm")
		plugins = append(plugins, PluginInfo{
			Name: name,
			Path: filepath.Join(dir, entry.Name()),
		})
	}
	return plugins, nil
}

// RegisterDiscoveredPlugins scans dir for .wasm files and registers each as a
// character factory in the character registry. Broken plugins are logged and skipped.
func RegisterDiscoveredPlugins(dir string) error {
	plugins, err := DiscoverPlugins(dir)
	if err != nil {
		return err
	}

	for _, plugin := range plugins {
		pluginPath := plugin.Path
		pluginName := plugin.Name

		character.Register(pluginName, func() character.Character {
			wasmBytes, err := os.ReadFile(pluginPath) // #nosec G304 — path comes from configured plugin directory
			if err != nil {
				log.Printf("WARNING: failed to read WASM plugin %s: %v", pluginPath, err)
				return character.NewNoOpCharacter()
			}

			canvas := NewFyneCanvasHost()
			host, err := LoadPlugin(wasmBytes, canvas)
			if err != nil {
				log.Printf("WARNING: failed to load WASM plugin %s: %v", pluginPath, err)
				return character.NewNoOpCharacter()
			}

			return host
		})
	}

	return nil
}
