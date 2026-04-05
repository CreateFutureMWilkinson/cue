package alert

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	timerBeepFrequency = 440.0
	timerBeepDuration  = 800
)

// MissedAlert represents a timer alert that was suppressed (e.g., during a meeting).
type MissedAlert struct {
	BlockType string
	TaskName  string
	Time      time.Time
	Message   string
}

// TimerAlertService manages audio alerts for timer events.
type TimerAlertService struct {
	soundPath   string
	volume      int
	beeper      Beeper
	fs          FileSystem
	audioPlayer AudioPlayer
	mu          sync.Mutex
}

// NewTimerAlertService creates a new TimerAlertService.
// beeper is always required. fileSystem is required when soundPath is non-empty.
// audioPlayer may be nil (fallback beep-only mode).
func NewTimerAlertService(soundPath string, volume int, beeper Beeper, fileSystem FileSystem, audioPlayer AudioPlayer) (*TimerAlertService, error) {
	if beeper == nil {
		return nil, fmt.Errorf("beeper is required")
	}
	if soundPath != "" && fileSystem == nil {
		return nil, fmt.Errorf("filesystem is required when sound path is set")
	}
	return &TimerAlertService{
		soundPath:   soundPath,
		volume:      clampVolume(volume),
		beeper:      beeper,
		fs:          fileSystem,
		audioPlayer: audioPlayer,
	}, nil
}

// PlayTimerEnd plays the timer-end alert or returns a MissedAlert if suppressed.
func (t *TimerAlertService) PlayTimerEnd(ctx context.Context, blockType string, taskName string, suppressed bool) (*MissedAlert, error) {
	if suppressed {
		return &MissedAlert{
			BlockType: blockType,
			TaskName:  taskName,
			Time:      time.Now(),
			Message:   formatBlockMessage(blockType, taskName),
		}, nil
	}

	t.mu.Lock()
	vol := t.volume
	t.mu.Unlock()

	if vol == 0 {
		return nil, nil
	}

	if t.soundPath != "" && t.audioPlayer != nil {
		go func() {
			if err := t.audioPlayer.PlayFile(t.soundPath, vol); err != nil {
				_ = t.beeper.Beep(timerBeepFrequency, timerBeepDuration)
			}
		}()
		return nil, nil
	}

	_ = t.beeper.Beep(timerBeepFrequency, timerBeepDuration)
	return nil, nil
}

// SetVolume sets the playback volume, clamped to 0-100.
func (t *TimerAlertService) SetVolume(volume int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.volume = clampVolume(volume)
}

// Volume returns the current playback volume.
func (t *TimerAlertService) Volume() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.volume
}

// formatBlockMessage creates a human-readable message for a missed alert.
func formatBlockMessage(blockType, taskName string) string {
	label := strings.ReplaceAll(blockType, "_", " ")
	if len(label) > 0 {
		label = strings.ToUpper(label[:1]) + label[1:]
	}
	return fmt.Sprintf("%s block ended: %s", label, taskName)
}
