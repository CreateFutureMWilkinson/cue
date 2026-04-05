package alert_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/alert"
	"github.com/stretchr/testify/suite"
)

// ---------------------------------------------------------------------------
// Suite
// ---------------------------------------------------------------------------

type TimerAlertServiceSuite struct {
	suite.Suite
}

func TestTimerAlertService(t *testing.T) {
	suite.Run(t, new(TimerAlertServiceSuite))
}

// ---------------------------------------------------------------------------
// Constructor Validation
// ---------------------------------------------------------------------------

func (s *TimerAlertServiceSuite) TestNilBeeperReturnsError() {
	svc, err := alert.NewTimerAlertService("/sounds/timer.wav", 75, nil, &mockFileSystem{}, &mockAudioPlayer{})
	s.Error(err)
	s.Nil(svc)
	s.Contains(err.Error(), "beeper")
}

func (s *TimerAlertServiceSuite) TestNilFileSystemWithSoundPathReturnsError() {
	svc, err := alert.NewTimerAlertService("/sounds/timer.wav", 75, &mockBeeper{}, nil, &mockAudioPlayer{})
	s.Error(err)
	s.Nil(svc)
	s.Contains(err.Error(), "filesystem")
}

func (s *TimerAlertServiceSuite) TestNilFileSystemWithEmptySoundPathSucceeds() {
	svc, err := alert.NewTimerAlertService("", 75, &mockBeeper{}, nil, &mockAudioPlayer{})
	s.NoError(err)
	s.NotNil(svc)
}

func (s *TimerAlertServiceSuite) TestValidArgsReturnService() {
	svc, err := alert.NewTimerAlertService("/sounds/timer.wav", 75, &mockBeeper{}, &mockFileSystem{}, &mockAudioPlayer{})
	s.NoError(err)
	s.NotNil(svc)
}

func (s *TimerAlertServiceSuite) TestAudioPlayerCanBeNil() {
	svc, err := alert.NewTimerAlertService("", 75, &mockBeeper{}, nil, nil)
	s.NoError(err)
	s.NotNil(svc)
}

// ---------------------------------------------------------------------------
// Play Behavior — Unsuppressed
// ---------------------------------------------------------------------------

func (s *TimerAlertServiceSuite) TestPlayTimerEndWithConfiguredSoundPlaysFile() {
	player := &mockAudioPlayer{}
	beeper := &mockBeeper{}
	fsys := &mockFileSystem{}

	svc, err := alert.NewTimerAlertService("/sounds/timer.wav", 75, beeper, fsys, player)
	s.Require().NoError(err)

	missed, err := svc.PlayTimerEnd(context.Background(), "focus", "Write report", false)
	s.NoError(err)
	s.Nil(missed, "unsuppressed play should return nil MissedAlert")

	// Wait briefly for async playback if applicable
	time.Sleep(100 * time.Millisecond)

	calls := player.getCalls()
	s.Require().Len(calls, 1)
	s.Equal("/sounds/timer.wav", calls[0].Path)
	s.Equal(75, calls[0].Volume)
}

func (s *TimerAlertServiceSuite) TestPlayTimerEndWithEmptySoundUseFallbackBeep() {
	player := &mockAudioPlayer{}
	beeper := &mockBeeper{}

	svc, err := alert.NewTimerAlertService("", 75, beeper, nil, player)
	s.Require().NoError(err)

	missed, err := svc.PlayTimerEnd(context.Background(), "focus", "Write report", false)
	s.NoError(err)
	s.Nil(missed)

	// Should use fallback beep, not audio player
	s.Equal(0, player.callCount(), "player should not be called when sound path is empty")
	s.Require().Equal(1, beeper.callCount(), "beeper should be called as fallback")

	call := beeper.lastCall()
	s.Equal(float64(440), call.FrequencyHz, "timer fallback should use 440Hz")
	s.Equal(800, call.DurationMs, "timer fallback should use 800ms")
}

func (s *TimerAlertServiceSuite) TestPlayTimerEndFileNotFoundFallsBackToBeep() {
	// FileSystem that reports file doesn't exist or player that fails
	player := &mockAudioPlayer{err: fmt.Errorf("file not found")}
	beeper := &mockBeeper{}
	fsys := &mockFileSystem{}

	svc, err := alert.NewTimerAlertService("/sounds/missing.wav", 75, beeper, fsys, player)
	s.Require().NoError(err)

	missed, err := svc.PlayTimerEnd(context.Background(), "focus", "Write report", false)
	s.NoError(err)
	s.Nil(missed)

	// Wait for async fallback if applicable
	time.Sleep(100 * time.Millisecond)

	s.GreaterOrEqual(beeper.callCount(), 1, "beeper should be called as fallback when file playback fails")
}

func (s *TimerAlertServiceSuite) TestPlayTimerEndAudioPlayerFailsFallsBackToBeep() {
	player := &mockAudioPlayer{err: fmt.Errorf("playback error")}
	beeper := &mockBeeper{}
	fsys := &mockFileSystem{}

	svc, err := alert.NewTimerAlertService("/sounds/timer.wav", 75, beeper, fsys, player)
	s.Require().NoError(err)

	missed, err := svc.PlayTimerEnd(context.Background(), "short_break", "Stretch", false)
	s.NoError(err)
	s.Nil(missed)

	// Wait for async fallback if applicable
	time.Sleep(100 * time.Millisecond)

	s.GreaterOrEqual(beeper.callCount(), 1, "beeper should be called when audio player fails")
}

// ---------------------------------------------------------------------------
// Play Behavior — Suppressed (Meeting)
// ---------------------------------------------------------------------------

func (s *TimerAlertServiceSuite) TestPlayTimerEndSuppressedDoesNotPlaySound() {
	player := &mockAudioPlayer{}
	beeper := &mockBeeper{}
	fsys := &mockFileSystem{}

	svc, err := alert.NewTimerAlertService("/sounds/timer.wav", 75, beeper, fsys, player)
	s.Require().NoError(err)

	missed, err := svc.PlayTimerEnd(context.Background(), "meeting", "Team standup", true)
	s.NoError(err)
	s.NotNil(missed, "suppressed play should return a MissedAlert")

	time.Sleep(100 * time.Millisecond)

	s.Equal(0, player.callCount(), "player should not be called when suppressed")
	s.Equal(0, beeper.callCount(), "beeper should not be called when suppressed")
}

func (s *TimerAlertServiceSuite) TestPlayTimerEndSuppressedReturnsMissedAlert() {
	beeper := &mockBeeper{}
	fsys := &mockFileSystem{}

	svc, err := alert.NewTimerAlertService("", 75, beeper, fsys, nil)
	s.Require().NoError(err)

	before := time.Now()
	missed, err := svc.PlayTimerEnd(context.Background(), "focus", "Deep work session", true)
	after := time.Now()

	s.NoError(err)
	s.Require().NotNil(missed)

	s.Equal("focus", missed.BlockType)
	s.Equal("Deep work session", missed.TaskName)
	s.True(!missed.Time.Before(before) && !missed.Time.After(after),
		"MissedAlert.Time should be approximately now")
	s.Contains(missed.Message, "Deep work session",
		"MissedAlert.Message should contain task name")
	s.Contains(missed.Message, "Focus block ended",
		"MissedAlert.Message should indicate block ended")
}

// ---------------------------------------------------------------------------
// Fallback Beep Tonality — Distinct from Notification
// ---------------------------------------------------------------------------

func (s *TimerAlertServiceSuite) TestTimerFallbackBeepDistinctFromNotification() {
	// Notification fallback: 1000Hz, 200ms (from defaultConfig in alert_test.go)
	// Timer fallback: 440Hz, 800ms (per design spec)
	// These must be different to allow auditory distinction.

	beeper := &mockBeeper{}
	svc, err := alert.NewTimerAlertService("", 75, beeper, nil, nil)
	s.Require().NoError(err)

	_, err = svc.PlayTimerEnd(context.Background(), "focus", "Task", false)
	s.NoError(err)

	s.Require().Equal(1, beeper.callCount())
	call := beeper.lastCall()

	// Timer beep values must differ from notification defaults (1000Hz, 200ms)
	s.Equal(float64(440), call.FrequencyHz, "timer beep frequency should be 440Hz")
	s.Equal(800, call.DurationMs, "timer beep duration should be 800ms")
	s.NotEqual(float64(1000), call.FrequencyHz, "timer beep must differ from notification frequency (1000Hz)")
	s.NotEqual(200, call.DurationMs, "timer beep must differ from notification duration (200ms)")
}

// ---------------------------------------------------------------------------
// Volume Control
// ---------------------------------------------------------------------------

func (s *TimerAlertServiceSuite) TestSetVolumeAndGet() {
	svc, err := alert.NewTimerAlertService("", 75, &mockBeeper{}, nil, nil)
	s.Require().NoError(err)

	svc.SetVolume(50)
	s.Equal(50, svc.Volume())
}

func (s *TimerAlertServiceSuite) TestSetVolumeClampsLow() {
	svc, err := alert.NewTimerAlertService("", 75, &mockBeeper{}, nil, nil)
	s.Require().NoError(err)

	svc.SetVolume(-10)
	s.Equal(0, svc.Volume())
}

func (s *TimerAlertServiceSuite) TestSetVolumeClampsHigh() {
	svc, err := alert.NewTimerAlertService("", 75, &mockBeeper{}, nil, nil)
	s.Require().NoError(err)

	svc.SetVolume(200)
	s.Equal(100, svc.Volume())
}

func (s *TimerAlertServiceSuite) TestVolumeZeroMutesPlayback() {
	player := &mockAudioPlayer{}
	beeper := &mockBeeper{}
	fsys := &mockFileSystem{}

	svc, err := alert.NewTimerAlertService("/sounds/timer.wav", 0, beeper, fsys, player)
	s.Require().NoError(err)

	missed, err := svc.PlayTimerEnd(context.Background(), "focus", "Task", false)
	s.NoError(err)
	s.Nil(missed)

	time.Sleep(100 * time.Millisecond)

	s.Equal(0, player.callCount(), "player should not be called when volume is 0")
	s.Equal(0, beeper.callCount(), "beeper should not be called when volume is 0")
}

func (s *TimerAlertServiceSuite) TestVolumeAppliedToFilePlayback() {
	player := &mockAudioPlayer{}
	beeper := &mockBeeper{}
	fsys := &mockFileSystem{}

	svc, err := alert.NewTimerAlertService("/sounds/timer.wav", 60, beeper, fsys, player)
	s.Require().NoError(err)

	svc.SetVolume(42)

	_, err = svc.PlayTimerEnd(context.Background(), "focus", "Task", false)
	s.NoError(err)

	time.Sleep(100 * time.Millisecond)

	calls := player.getCalls()
	s.Require().Len(calls, 1)
	s.Equal(42, calls[0].Volume, "volume passed to PlayFile should match SetVolume value")
}

// ---------------------------------------------------------------------------
// MissedAlert Struct Fields
// ---------------------------------------------------------------------------

func (s *TimerAlertServiceSuite) TestMissedAlertBlockTypeMatchesInput() {
	svc, err := alert.NewTimerAlertService("", 75, &mockBeeper{}, nil, nil)
	s.Require().NoError(err)

	missed, err := svc.PlayTimerEnd(context.Background(), "long_break", "Lunch walk", true)
	s.NoError(err)
	s.Require().NotNil(missed)
	s.Equal("long_break", missed.BlockType)
}

func (s *TimerAlertServiceSuite) TestMissedAlertTaskNameMatchesInput() {
	svc, err := alert.NewTimerAlertService("", 75, &mockBeeper{}, nil, nil)
	s.Require().NoError(err)

	missed, err := svc.PlayTimerEnd(context.Background(), "focus", "Code review PR #42", true)
	s.NoError(err)
	s.Require().NotNil(missed)
	s.Equal("Code review PR #42", missed.TaskName)
}

func (s *TimerAlertServiceSuite) TestMissedAlertTimeIsSet() {
	svc, err := alert.NewTimerAlertService("", 75, &mockBeeper{}, nil, nil)
	s.Require().NoError(err)

	before := time.Now()
	missed, err := svc.PlayTimerEnd(context.Background(), "focus", "Task", true)
	after := time.Now()

	s.NoError(err)
	s.Require().NotNil(missed)
	s.False(missed.Time.IsZero(), "MissedAlert.Time should be set")
	s.True(!missed.Time.Before(before) && !missed.Time.After(after))
}

func (s *TimerAlertServiceSuite) TestMissedAlertMessageFormatted() {
	svc, err := alert.NewTimerAlertService("", 75, &mockBeeper{}, nil, nil)
	s.Require().NoError(err)

	missed, err := svc.PlayTimerEnd(context.Background(), "focus", "Write report", true)
	s.NoError(err)
	s.Require().NotNil(missed)
	s.Equal("Focus block ended: Write report", missed.Message)
}
