package main

import (
	"log"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	characteruat "github.com/CreateFutureMWilkinson/cue/cmd/character-uat"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/character/fairy"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/character/wasmhost"
)

func init() {
	character.Register("fairy", func() character.Character {
		f := fairy.NewFairyCharacter()
		f.SetRefreshHook(func() {
			fyne.Do(func() { f.Widget().Refresh() })
		})
		return f
	})

	// Discover WASM character plugins from ~/.cue/characters/.
	home, err := os.UserHomeDir()
	if err == nil {
		charDir := filepath.Join(home, ".cue", "characters")
		if err := wasmhost.RegisterDiscoveredPlugins(charDir); err != nil {
			log.Printf("warning: WASM plugin discovery failed: %v", err)
		}
	}
}

func main() {
	a := app.New()
	w := characteruat.NewUATWindow(a)
	w.Run()
}
