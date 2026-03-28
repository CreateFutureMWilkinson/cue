package character

import (
	"fmt"
	"sync"

	"fyne.io/fyne/v2"
)

const (
	// NoneCharacterName is the registry name for the no-op character.
	NoneCharacterName = "none"
)

// Character defines the interface for an animated character in the UI.
type Character interface {
	Name() string
	TransitionTo(state CharacterState)
	CurrentState() CharacterState
	Widget() fyne.CanvasObject
}

// CharacterFactory is a constructor function that creates a Character.
type CharacterFactory func() Character

var (
	registryMu sync.Mutex
	registry   = map[string]CharacterFactory{}
)

func init() {
	registry[NoneCharacterName] = func() Character { return NewNoOpCharacter() }
}

// Register adds a named character factory to the registry.
func Register(name string, factory CharacterFactory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = factory
}

// Create instantiates a character by its registered name.
func Create(name string) (Character, error) {
	registryMu.Lock()
	defer registryMu.Unlock()
	factory, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("character %q not registered", name)
	}
	return factory(), nil
}

// Available returns the names of all registered characters.
func Available() []string {
	registryMu.Lock()
	defer registryMu.Unlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// ResetRegistry clears the registry and re-registers the default "none" character.
func ResetRegistry() {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = map[string]CharacterFactory{
		NoneCharacterName: func() Character { return NewNoOpCharacter() },
	}
}
