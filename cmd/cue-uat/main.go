package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	characteruat "github.com/CreateFutureMWilkinson/cue/cmd/character-uat"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/character/fairy"
)

func init() {
	character.Register("fairy", func() character.Character {
		f := fairy.NewFairyCharacter()
		f.SetRefreshHook(func() {
			fyne.Do(func() { f.Widget().Refresh() })
		})
		return f
	})
}

func main() {
	a := app.New()
	w := characteruat.NewUATWindow(a)
	w.Run()
}
