package main

import (
	"fyne.io/fyne/v2/app"

	characteruat "github.com/CreateFutureMWilkinson/cue/cmd/character-uat"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
)

func init() {
	character.Register("fairy", func() character.Character {
		return character.NewFairyCharacter()
	})
}

func main() {
	a := app.New()
	w := characteruat.NewUATWindow(a)
	w.Run()
}
