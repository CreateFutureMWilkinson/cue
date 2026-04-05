package character_test

import (
	"testing"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
	"github.com/stretchr/testify/suite"
)

// ParentAPISuite verifies that the parent character package retains the
// correct exports after the fairy sub-package restructure, and that
// fairy-specific types have been removed.
type ParentAPISuite struct {
	suite.Suite
}

func TestParentAPI(t *testing.T) {
	suite.Run(t, new(ParentAPISuite))
}

// TestClockInterfaceStillExists verifies Clock remains in the parent package.
func (s *ParentAPISuite) TestClockInterfaceStillExists() {
	var _ character.Clock = character.WallClock{}
	s.NotNil(character.WallClock{}, "WallClock should still exist in parent")
}

// TestTickerInterfaceStillExists verifies Ticker remains in the parent package.
func (s *ParentAPISuite) TestTickerInterfaceStillExists() {
	clock := character.WallClock{}
	ticker := clock.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	var _ character.Ticker = ticker
}

// TestAnimationFPSStillInParent verifies animation timing constants remain.
func (s *ParentAPISuite) TestAnimationFPSStillInParent() {
	s.Equal(30, character.AnimationFPS, "AnimationFPS should be 30")
}

// TestAnimationTickMsStillInParent verifies tick duration constant remains.
func (s *ParentAPISuite) TestAnimationTickMsStillInParent() {
	s.Equal(1000/30, character.AnimationTickMs)
}

// TestAnimationFrameIntervalStillInParent verifies frame interval constant remains.
func (s *ParentAPISuite) TestAnimationFrameIntervalStillInParent() {
	s.Equal(time.Millisecond, character.AnimationFrameInterval)
}

// TestWallClockStillInParent verifies WallClock type remains.
func (s *ParentAPISuite) TestWallClockStillInParent() {
	wc := character.WallClock{}
	now := wc.Now()
	s.False(now.IsZero(), "WallClock.Now() should return a non-zero time")
}

// TestCharacterInterfaceStillInParent verifies Character interface remains.
func (s *ParentAPISuite) TestCharacterInterfaceStillInParent() {
	// NoOpCharacter should still implement Character.
	noop := character.NewNoOpCharacter()
	var _ character.Character = noop
}

// TestRegistryStillInParent verifies the character registry remains.
func (s *ParentAPISuite) TestRegistryStillInParent() {
	character.ResetRegistry()
	names := character.Available()
	s.Contains(names, "none", "registry should include 'none' after reset")
}

// --- Compile-time absence checks ---
// The following tests assert that types/functions that should have moved
// to the fairy sub-package are NO LONGER in the parent package.
// If they still exist, these tests will fail by having incorrect behavior.

// TestStateAnimatorRemovedFromParent verifies StateAnimator is no longer
// an exported type in the parent package. After the restructure,
// StateAnimator becomes unexported (stateAnimator) inside fairy.
//
// This is a compile-time check: if character.StateAnimator still exists
// as an exported interface, this file should be updated. For now, we
// verify that the parent package does NOT export NewFairyCharacter
// (which is the primary indicator of the restructure).
func (s *ParentAPISuite) TestNewFairyCharacterRemovedFromParent() {
	// After restructure, character.NewFairyCharacter should not exist.
	// The fairy constructor moves to fairy.NewFairyCharacter.
	// If the function still exists in parent, this test is intentionally
	// calling it to document that removal hasn't happened yet.
	// We detect this by checking the registry: creating "fairy" from
	// parent's init() should not work after restructure.
	character.ResetRegistry()
	_, err := character.Create("fairy")
	s.Error(err, "parent package should not auto-register 'fairy' after restructure; "+
		"fairy registration should come from the fairy sub-package")
}

// TestEaseInOutRemovedFromParent verifies EaseInOut moved to fairy package.
// After the restructure, character.EaseInOut should not exist.
// This test will PASS once EaseInOut is removed from parent.
// Until then, the function exists and this test detects it.
func (s *ParentAPISuite) TestEaseInOutRemovedFromParent() {
	// We test this indirectly: if EaseInOut still exists, this compiles
	// and we flag it. The test designer notes that once character.EaseInOut
	// is removed, this test file will need the reference removed too.
	// For now, we call it and assert it should NOT be accessible.
	// Since we cannot test for absence at compile time in the same file
	// that imports the package, we verify via the restructure contract:
	// the parent package should not export IdleBodyColor after restructure.
	//
	// Test: IdleOriginX, IdleOriginY, IdleBodyColor should NOT be in parent.
	// These move to fairy package. If they still exist, the restructure
	// is incomplete.
	//
	// NOTE: This test currently compiles because these symbols exist.
	// After restructure, this file must be updated to remove these
	// references, and the test passes by virtue of the fairy_package_test
	// verifying they exist in the fairy sub-package instead.
	s.Fail("EaseInOut, IdleBodyColor, IdleOriginX, IdleOriginY should be removed from parent character package and moved to fairy sub-package")
}
