package fairy_test

import (
	"image/color"
	"math/rand"
	"testing"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/character/fairy"
	"github.com/stretchr/testify/suite"
)

// FairyPackageSuite verifies the fairy sub-package exports.
type FairyPackageSuite struct {
	suite.Suite
}

func TestFairyPackage(t *testing.T) {
	suite.Run(t, new(FairyPackageSuite))
}

func (s *FairyPackageSuite) TestNewFairyCharacterReturnsNonNil() {
	fc := fairy.NewFairyCharacter()
	s.NotNil(fc, "NewFairyCharacter() should return a non-nil character")
}

func (s *FairyPackageSuite) TestNewFairyCharacterImplementsCharacter() {
	fc := fairy.NewFairyCharacter()
	// Verify it satisfies the Character interface from the parent package.
	var _ character.Character = fc
}

func (s *FairyPackageSuite) TestIdleBodyColorValue() {
	expected := color.RGBA{R: 0x00, G: 0xFF, B: 0x00, A: 0xFF}
	s.Equal(expected, fairy.IdleBodyColor,
		"IdleBodyColor should be bright green (#00FF00)")
}

func (s *FairyPackageSuite) TestIdleOriginConstants() {
	s.Equal(0.5, fairy.IdleOriginX, "IdleOriginX should be 0.5")
	s.Equal(1.0, fairy.IdleOriginY, "IdleOriginY should be 1.0")
}

func (s *FairyPackageSuite) TestIdleBreathCycleConstant() {
	s.Equal(3.0, fairy.IdleBreathCycleSec,
		"IdleBreathCycleSec should be 3.0")
}

func (s *FairyPackageSuite) TestIdleGlowMinMaxConstants() {
	s.Equal(0.3, fairy.IdleGlowMin, "IdleGlowMin should be 0.3")
	s.Equal(0.8, fairy.IdleGlowMax, "IdleGlowMax should be 0.8")
}

func (s *FairyPackageSuite) TestNewIdleAnimatorExists() {
	clock := character.WallClock{}
	animator := fairy.NewIdleAnimator(clock)
	s.NotNil(animator, "NewIdleAnimator should return a non-nil animator")
}

func (s *FairyPackageSuite) TestNewWorkingAnimatorExists() {
	clock := character.WallClock{}
	animator := fairy.NewWorkingAnimator(clock)
	s.NotNil(animator, "NewWorkingAnimator should return a non-nil animator")
}

func (s *FairyPackageSuite) TestNewStartupAnimatorExists() {
	clock := character.WallClock{}
	animator := fairy.NewStartupAnimator(clock, func() {})
	s.NotNil(animator, "NewStartupAnimator should return a non-nil animator")
}

func (s *FairyPackageSuite) TestNewNotifyAnimatorExists() {
	clock := character.WallClock{}
	rng := rand.New(rand.NewSource(42)) // #nosec G404 -- test-only
	animator := fairy.NewNotifyAnimator(clock, rng)
	s.NotNil(animator, "NewNotifyAnimator should return a non-nil animator")
}

func (s *FairyPackageSuite) TestNewErrorAnimatorExists() {
	clock := character.WallClock{}
	animator := fairy.NewErrorAnimator(clock)
	s.NotNil(animator, "NewErrorAnimator should return a non-nil animator")
}

func (s *FairyPackageSuite) TestNewShutdownAnimatorExists() {
	clock := character.WallClock{}
	animator := fairy.NewShutdownAnimator(clock)
	s.NotNil(animator, "NewShutdownAnimator should return a non-nil animator")
}

func (s *FairyPackageSuite) TestEaseInOutExists() {
	// EaseInOut should be exported from the fairy package (moved from parent).
	result := fairy.EaseInOut(0.5)
	s.InDelta(0.5, result, 0.001,
		"EaseInOut(0.5) should return approximately 0.5")
}

func (s *FairyPackageSuite) TestEaseInOutBoundaryValues() {
	s.InDelta(0.0, fairy.EaseInOut(0.0), 0.001, "EaseInOut(0) should be 0")
	s.InDelta(1.0, fairy.EaseInOut(1.0), 0.001, "EaseInOut(1) should be 1")
}

func (s *FairyPackageSuite) TestIdleGlowIntensityExists() {
	// IdleGlowIntensity should be exported from fairy package.
	result := fairy.IdleGlowIntensity(0.0)
	s.InDelta(0.55, result, 0.01,
		"IdleGlowIntensity(0) should return the midpoint of glow range")
}

func (s *FairyPackageSuite) TestAnimationFPSNotInFairyPackage() {
	// AnimationFPS should remain in the parent character package, not fairy.
	// This is a compile-time assertion: we access it from parent.
	_ = character.AnimationFPS
	// If fairy.AnimationFPS existed, it would be a design error.
	// We cannot test for absence at compile time, so we just verify the parent has it.
}

func (s *FairyPackageSuite) TestFairyNameReturnsFairy() {
	fc := fairy.NewFairyCharacter()
	s.Equal("fairy", fc.Name(), "fairy character Name() should return 'fairy'")
}

func (s *FairyPackageSuite) TestFairyStartsInIdleState() {
	fc := fairy.NewFairyCharacter()
	s.Equal(character.StateIdle, fc.CurrentState(),
		"newly created fairy should start in StateIdle")
}
