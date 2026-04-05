package character

// CharacterState represents the current animation/behavioral state of a character.
type CharacterState int

const (
	StateIdle CharacterState = iota
	StateStarting
	StateWorking
	StateNotifying
	StateError
	StateShuttingDown
)

var stateNames = [...]string{
	"Idle",
	"Starting",
	"Working",
	"Notifying",
	"Error",
	"ShuttingDown",
}

// String returns the human-readable name of the state.
func (s CharacterState) String() string {
	if int(s) < len(stateNames) {
		return stateNames[s]
	}
	return "Unknown"
}
