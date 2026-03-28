package alert

import (
	"context"
	"fmt"
	"io/fs"
	"math/rand/v2"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gen2brain/beeep"
)

// supportedExtensions lists audio file extensions that can be played.
var supportedExtensions = map[string]bool{
	".mp3": true,
	".wav": true,
	".ogg": true,
}

// AlertConfig holds configuration for the alert service.
type AlertConfig struct {
	AudioEnabled         bool
	AudioDir             string
	AudioCooldownSeconds int
	AudioVolume          int
	FallbackFrequency    int
	FallbackDurationMs   int
}

// Beeper abstracts audio beep functionality.
type Beeper interface {
	Beep(frequencyHz float64, durationMs int) error
}

// FileSystem abstracts directory reading.
type FileSystem interface {
	ReadDir(path string) ([]fs.DirEntry, error)
}

// AudioPlayer abstracts audio file playback.
type AudioPlayer interface {
	PlayFile(path string, volume int) error
}

// beeepBeeper wraps the beeep package for production use.
type beeepBeeper struct{}

func (b *beeepBeeper) Beep(frequencyHz float64, durationMs int) error {
	return beeep.Beep(frequencyHz, durationMs)
}

// NewBeeepBeeper creates a new production beeper using the beeep package.
func NewBeeepBeeper() Beeper {
	return &beeepBeeper{}
}

// AlertService manages audio alerts with cooldown support.
type AlertService struct {
	cfg       AlertConfig
	beeper    Beeper
	fs        FileSystem
	player    AudioPlayer
	volume    int
	lastAlert time.Time
	mu        sync.Mutex
	now       func() time.Time
}

// NewAlertService creates a new AlertService, validating dependencies.
func NewAlertService(cfg AlertConfig, beeper Beeper, fsys FileSystem, player AudioPlayer) (*AlertService, error) {
	if beeper == nil {
		return nil, fmt.Errorf("beeper is required")
	}
	if fsys == nil {
		return nil, fmt.Errorf("filesystem is required")
	}
	return &AlertService{
		cfg:    cfg,
		beeper: beeper,
		fs:     fsys,
		player: player,
		volume: cfg.AudioVolume,
		now:    time.Now,
	}, nil
}

// SetNowFunc allows tests to inject a clock function for cooldown testing.
func (a *AlertService) SetNowFunc(fn func() time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.now = fn
}

// SetVolume sets the playback volume, clamped to 0-100.
func (a *AlertService) SetVolume(volume int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if volume < 0 {
		volume = 0
	}
	if volume > 100 {
		volume = 100
	}
	a.volume = volume
}

// PlayNotification plays a notification alert with configurable cooldown.
func (a *AlertService) PlayNotification(ctx context.Context) error {
	if !a.cfg.AudioEnabled {
		return nil
	}

	a.mu.Lock()
	now := a.now()
	cooldown := time.Duration(a.cfg.AudioCooldownSeconds) * time.Second
	if !a.lastAlert.IsZero() && now.Sub(a.lastAlert) < cooldown {
		a.mu.Unlock()
		return nil
	}
	a.lastAlert = now
	volume := a.volume
	a.mu.Unlock()

	// Try file playback.
	if a.cfg.AudioDir == "" || a.player == nil {
		return a.fallbackBeep()
	}

	entries, err := a.fs.ReadDir(a.cfg.AudioDir)
	if err != nil {
		return a.fallbackBeep()
	}

	var supported []string
	for _, entry := range entries {
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if supportedExtensions[ext] {
			supported = append(supported, entry.Name())
		}
	}

	if len(supported) == 0 {
		return a.fallbackBeep()
	}

	chosen := supported[rand.IntN(len(supported))]
	path := filepath.Join(a.cfg.AudioDir, chosen)

	go func() {
		if playErr := a.player.PlayFile(path, volume); playErr != nil {
			_ = a.fallbackBeep()
		}
	}()

	return nil
}

// fallbackBeep plays the configured fallback tone via the beeper.
func (a *AlertService) fallbackBeep() error {
	return a.beeper.Beep(float64(a.cfg.FallbackFrequency), a.cfg.FallbackDurationMs)
}
