package character_test

import (
	"testing"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
	"github.com/stretchr/testify/suite"
)

type CharacterRegistrySuite struct {
	suite.Suite
}

func TestCharacterRegistry(t *testing.T) {
	suite.Run(t, new(CharacterRegistrySuite))
}

func (s *CharacterRegistrySuite) SetupTest() {
	character.ResetRegistry()
}

func (s *CharacterRegistrySuite) TestRegisterAndCreateCharacterByName() {
	character.Register("test-char", func() character.Character {
		return character.NewNoOpCharacter()
	})

	char, err := character.Create("test-char")
	s.Require().NoError(err)
	s.Require().NotNil(char)
	s.Equal("none", char.Name()) // NoOpCharacter returns "none"
}

func (s *CharacterRegistrySuite) TestCreateReturnsErrorForUnregisteredName() {
	_, err := character.Create("nonexistent")
	s.Error(err)
}

func (s *CharacterRegistrySuite) TestAvailableListsAllRegisteredNamesIncludingNone() {
	available := character.Available()
	s.Contains(available, "none")
}

func (s *CharacterRegistrySuite) TestAvailableIncludesNewlyRegisteredCharacter() {
	character.Register("custom-fairy", func() character.Character {
		return character.NewNoOpCharacter()
	})

	available := character.Available()
	s.Contains(available, "none")
	s.Contains(available, "custom-fairy")
}
