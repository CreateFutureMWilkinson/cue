package alert

import (
	"context"
	"crypto/rand"
	"fmt"
	"io/fs"
	"math/big"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gen2brain/beeep"
)

const (
	// Volume limits for audio playback
	minVolume = 0
	maxVolume = 100
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
	a.volume = clampVolume(volume)
}

// clampVolume ensures volume is within the valid range.
func clampVolume(volume int) int {
	if volume < minVolume {
		return minVolume
	}
	if volume > maxVolume {
		return maxVolume
	}
	return volume
}

// PlayNotification plays a notification alert with configurable cooldown.
// File playback runs asynchronously to avoid blocking; if playback fails,
// it falls back to a beep tone.
func (a *AlertService) PlayNotification(ctx context.Context) error {
	if !a.cfg.AudioEnabled {
		return nil
	}

	// Check cooldown and update last alert time under lock
	if !a.shouldPlayAlert() {
		return nil
	}

	// Try file playback first, fall back to beep if unavailable
	audioFile, volume, err := a.selectAudioFile()
	if err != nil {
		return a.fallbackBeep()
	}

	// Play audio file asynchronously to avoid blocking
	// If file playback fails, fallback beep runs synchronously
	go func() {
		if playErr := a.player.PlayFile(audioFile, volume); playErr != nil {
			_ = a.fallbackBeep()
		}
	}()

	return nil
}

// shouldPlayAlert checks cooldown and updates last alert time if allowed.
// Returns true if alert should play, false if still in cooldown.
func (a *AlertService) shouldPlayAlert() bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := a.now()
	cooldown := time.Duration(a.cfg.AudioCooldownSeconds) * time.Second
	if !a.lastAlert.IsZero() && now.Sub(a.lastAlert) < cooldown {
		return false
	}

	a.lastAlert = now
	return true
}

// selectAudioFile chooses a random supported audio file from the configured directory.
// Returns the full file path, current volume setting, and any error encountered.
func (a *AlertService) selectAudioFile() (string, int, error) {
	if a.cfg.AudioDir == "" || a.player == nil {
		return "", 0, fmt.Errorf("audio directory or player not configured")
	}

	entries, err := a.fs.ReadDir(a.cfg.AudioDir)
	if err != nil {
		return "", 0, fmt.Errorf("reading audio directory: %w", err)
	}

	supportedFiles := a.filterSupportedAudioFiles(entries)
	if len(supportedFiles) == 0 {
		return "", 0, fmt.Errorf("no supported audio files found")
	}

	randomIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(supportedFiles))))
	if err != nil {
		return "", 0, fmt.Errorf("generating secure random index for audio file selection: %w", err)
	}
	selectedFile := supportedFiles[randomIndex.Int64()]
	path := filepath.Join(a.cfg.AudioDir, selectedFile)

	a.mu.Lock()
	volume := a.volume
	a.mu.Unlock()

	return path, volume, nil
}

// filterSupportedAudioFiles returns filenames with supported audio extensions.
// Rejects path traversal attempts and directories for security.
func (a *AlertService) filterSupportedAudioFiles(entries []fs.DirEntry) []string {
	cleanDir := filepath.Clean(a.cfg.AudioDir)
	var supported []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		if !isSafeAudioFile(cleanDir, filename) {
			continue
		}

		ext := strings.ToLower(filepath.Ext(filename))
		if supportedExtensions[ext] {
			supported = append(supported, filename)
		}
	}
	return supported
}

// isSafeAudioFile validates that a filename is safe from path traversal attacks.
func isSafeAudioFile(audioDir, filename string) bool {
	resolvedPath := filepath.Clean(filepath.Join(audioDir, filename))
	expectedPrefix := audioDir + string(filepath.Separator)
	return strings.HasPrefix(resolvedPath, expectedPrefix)
}

// fallbackBeep plays the configured fallback tone via the beeper.
func (a *AlertService) fallbackBeep() error {
	return a.beeper.Beep(float64(a.cfg.FallbackFrequency), a.cfg.FallbackDurationMs)
}
