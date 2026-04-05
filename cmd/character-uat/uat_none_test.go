package characteruat_test

import (
	"testing"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
	"github.com/stretchr/testify/suite"
)

// UATNoneSuite verifies the "none" character is available in the UAT dropdown.
type UATNoneSuite struct {
	suite.Suite
}

func TestUATNone(t *testing.T) {
	suite.Run(t, new(UATNoneSuite))
}

func (s *UATNoneSuite) SetupTest() {
	character.ResetRegistry()
}

// TestAvailableCharacterNamesIncludesNone verifies that the UAT character
// dropdown includes "none" as a selectable option. Currently,
// availableCharacterNames() filters out NoneCharacterName, but after the
// restructure it should be included so users can select no character.
func (s *UATNoneSuite) TestAvailableCharacterNamesIncludesNone() {
	// The "none" character is always registered via init() in the parent package.
	names := character.Available()
	s.Contains(names, "none", "registry should contain 'none'")

	// The UAT's availableCharacterNames() currently EXCLUDES "none".
	// After Feature 041, it should INCLUDE "none" in the dropdown.
	// We cannot call the unexported function directly, so we test the
	// contract: "none" should be a valid, createable character that
	// appears in Available() AND is not filtered out by the UAT.
	//
	// To verify the UAT filtering change, we check that creating "none"
	// returns a valid no-op character, and assert the design requirement
	// that "none" should appear in the dropdown.
	ch, err := character.Create("none")
	s.Require().NoError(err, "creating 'none' character should succeed")
	s.Equal("none", ch.Name())

	// This assertion documents the Feature 041 requirement:
	// availableCharacterNames() must include "none".
	// Since we cannot call the unexported function from _test package,
	// we mark this as a known failing test that the implementer must
	// address by changing the filter logic.
	s.Fail("availableCharacterNames() should include 'none' in the character dropdown after Feature 041 restructure")
}
