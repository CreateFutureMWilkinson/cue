package alert_test

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/alert"
	"github.com/stretchr/testify/suite"
)

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

type beepCall struct {
	FrequencyHz float64
	DurationMs  int
}

type mockBeeper struct {
	mu    sync.Mutex
	calls []beepCall
	err   error
}

func (m *mockBeeper) Beep(frequencyHz float64, durationMs int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, beepCall{FrequencyHz: frequencyHz, DurationMs: durationMs})
	return m.err
}

func (m *mockBeeper) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

func (m *mockBeeper) lastCall() beepCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls[len(m.calls)-1]
}

// mockDirEntry implements fs.DirEntry for testing.
type mockDirEntry struct {
	name  string
	isDir bool
}

func (e *mockDirEntry) Name() string               { return e.name }
func (e *mockDirEntry) IsDir() bool                { return e.isDir }
func (e *mockDirEntry) Type() fs.FileMode          { return 0 }
func (e *mockDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

// mockFileSystem implements alert.FileSystem.
type mockFileSystem struct {
	entries []fs.DirEntry
	err     error
}

func (m *mockFileSystem) ReadDir(path string) ([]fs.DirEntry, error) {
	return m.entries, m.err
}

// mockAudioPlayer implements alert.AudioPlayer.
type mockAudioPlayer struct {
	mu    sync.Mutex
	calls []playerCall
	err   error
	delay time.Duration
}

type playerCall struct {
	Path   string
	Volume int
}

func (m *mockAudioPlayer) PlayFile(path string, volume int) error {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, playerCall{Path: path, Volume: volume})
	return m.err
}

func (m *mockAudioPlayer) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

func (m *mockAudioPlayer) getCalls() []playerCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]playerCall, len(m.calls))
	copy(result, m.calls)
	return result
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func entries(names ...string) []fs.DirEntry {
	result := make([]fs.DirEntry, len(names))
	for i, n := range names {
		result[i] = &mockDirEntry{name: n}
	}
	return result
}

func defaultConfig() alert.AlertConfig {
	return alert.AlertConfig{
		AudioEnabled:         true,
		AudioDir:             "/sounds",
		AudioCooldownSeconds: 2,
		AudioVolume:          80,
		FallbackFrequency:    1000,
		FallbackDurationMs:   200,
	}
}

// ---------------------------------------------------------------------------
// Suite
// ---------------------------------------------------------------------------

type AlertSuite struct {
	suite.Suite
}

func TestAlert(t *testing.T) {
	suite.Run(t, new(AlertSuite))
}

// ---------------------------------------------------------------------------
// Constructor Tests
// ---------------------------------------------------------------------------

func (s *AlertSuite) TestNewAlertServiceRequiresBeeper() {
	svc, err := alert.NewAlertService(defaultConfig(), nil, &mockFileSystem{}, &mockAudioPlayer{})
	s.Error(err)
	s.Nil(svc)
	s.Contains(err.Error(), "beeper")
}

func (s *AlertSuite) TestNewAlertServiceRequiresFileSystem() {
	svc, err := alert.NewAlertService(defaultConfig(), &mockBeeper{}, nil, &mockAudioPlayer{})
	s.Error(err)
	s.Nil(svc)
	s.Contains(err.Error(), "filesystem")
}

func (s *AlertSuite) TestNewAlertServiceAllowsNilPlayer() {
	svc, err := alert.NewAlertService(defaultConfig(), &mockBeeper{}, &mockFileSystem{}, nil)
	s.NoError(err)
	s.NotNil(svc)
}

func (s *AlertSuite) TestNewAlertServiceAcceptsAllDeps() {
	svc, err := alert.NewAlertService(defaultConfig(), &mockBeeper{}, &mockFileSystem{}, &mockAudioPlayer{})
	s.NoError(err)
	s.NotNil(svc)
}

// ---------------------------------------------------------------------------
// PlayNotification - File Playback
// ---------------------------------------------------------------------------

func (s *AlertSuite) TestPlayNotificationPlaysRandomFile() {
	mp3Files := entries("alert1.mp3", "alert2.mp3", "alert3.mp3")
	fsys := &mockFileSystem{entries: mp3Files}
	player := &mockAudioPlayer{}
	cfg := defaultConfig()
	cfg.AudioVolume = 70

	svc, err := alert.NewAlertService(cfg, &mockBeeper{}, fsys, player)
	s.Require().NoError(err)

	err = svc.PlayNotification(context.Background())
	s.NoError(err)

	// Wait briefly for async playback
	time.Sleep(100 * time.Millisecond)

	calls := player.getCalls()
	s.Require().Len(calls, 1)
	s.Equal(70, calls[0].Volume)

	// File should be one of the three mp3 files
	validFiles := map[string]bool{
		"/sounds/alert1.mp3": true,
		"/sounds/alert2.mp3": true,
		"/sounds/alert3.mp3": true,
	}
	s.True(validFiles[calls[0].Path], "unexpected file: %s", calls[0].Path)
}

func (s *AlertSuite) TestPlayNotificationFiltersUnsupportedFiles() {
	mixedFiles := entries("readme.txt", "notes.pdf", "alert.mp3")
	fsys := &mockFileSystem{entries: mixedFiles}
	player := &mockAudioPlayer{}

	svc, err := alert.NewAlertService(defaultConfig(), &mockBeeper{}, fsys, player)
	s.Require().NoError(err)

	// Call multiple times (advancing past cooldown each time) to ensure only .mp3 is selected
	for i := 0; i < 10; i++ {
		svc.SetNowFunc(func() time.Time {
			return time.Now().Add(time.Duration(i*3) * time.Second)
		})
		_ = svc.PlayNotification(context.Background())
	}

	time.Sleep(100 * time.Millisecond)

	for _, call := range player.getCalls() {
		s.Equal("/sounds/alert.mp3", call.Path)
	}
}

func (s *AlertSuite) TestPlayNotificationSupportsAllFormats() {
	allFormats := entries("a.mp3", "b.wav", "c.ogg")
	fsys := &mockFileSystem{entries: allFormats}
	player := &mockAudioPlayer{}

	svc, err := alert.NewAlertService(defaultConfig(), &mockBeeper{}, fsys, player)
	s.Require().NoError(err)

	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		svc.SetNowFunc(func() time.Time {
			return time.Now().Add(time.Duration(i*3) * time.Second)
		})
		_ = svc.PlayNotification(context.Background())
	}

	time.Sleep(200 * time.Millisecond)

	for _, call := range player.getCalls() {
		seen[call.Path] = true
	}

	s.True(seen["/sounds/a.mp3"], "mp3 file never selected")
	s.True(seen["/sounds/b.wav"], "wav file never selected")
	s.True(seen["/sounds/c.ogg"], "ogg file never selected")
}

func (s *AlertSuite) TestPlayNotificationAsyncDoesNotBlock() {
	fsys := &mockFileSystem{entries: entries("slow.mp3")}
	player := &mockAudioPlayer{delay: 2 * time.Second}

	svc, err := alert.NewAlertService(defaultConfig(), &mockBeeper{}, fsys, player)
	s.Require().NoError(err)

	start := time.Now()
	err = svc.PlayNotification(context.Background())
	elapsed := time.Since(start)

	s.NoError(err)
	s.Less(elapsed, 500*time.Millisecond, "PlayNotification should return immediately, took %v", elapsed)
}

// ---------------------------------------------------------------------------
// PlayNotification - Fallback
// ---------------------------------------------------------------------------

func (s *AlertSuite) TestPlayNotificationFallbackWhenNoDir() {
	beeper := &mockBeeper{}
	cfg := defaultConfig()
	cfg.AudioDir = "" // no dir

	svc, err := alert.NewAlertService(cfg, beeper, &mockFileSystem{}, &mockAudioPlayer{})
	s.Require().NoError(err)

	err = svc.PlayNotification(context.Background())
	s.NoError(err)

	s.Require().Equal(1, beeper.callCount())
}

func (s *AlertSuite) TestPlayNotificationFallbackWhenDirEmpty() {
	beeper := &mockBeeper{}
	fsys := &mockFileSystem{entries: []fs.DirEntry{}}

	svc, err := alert.NewAlertService(defaultConfig(), beeper, fsys, &mockAudioPlayer{})
	s.Require().NoError(err)

	err = svc.PlayNotification(context.Background())
	s.NoError(err)

	s.Require().Equal(1, beeper.callCount())
}

func (s *AlertSuite) TestPlayNotificationFallbackWhenNoSupportedFiles() {
	beeper := &mockBeeper{}
	fsys := &mockFileSystem{entries: entries("readme.txt", "data.csv")}

	svc, err := alert.NewAlertService(defaultConfig(), beeper, fsys, &mockAudioPlayer{})
	s.Require().NoError(err)

	err = svc.PlayNotification(context.Background())
	s.NoError(err)

	s.Require().Equal(1, beeper.callCount())
}

func (s *AlertSuite) TestPlayNotificationFallbackWhenPlayerNil() {
	beeper := &mockBeeper{}
	fsys := &mockFileSystem{entries: entries("alert.mp3")}

	svc, err := alert.NewAlertService(defaultConfig(), beeper, fsys, nil)
	s.Require().NoError(err)

	err = svc.PlayNotification(context.Background())
	s.NoError(err)

	s.Require().Equal(1, beeper.callCount())
}

func (s *AlertSuite) TestPlayNotificationFallbackUsesConfiguredTone() {
	beeper := &mockBeeper{}
	cfg := defaultConfig()
	cfg.AudioDir = ""
	cfg.FallbackFrequency = 440
	cfg.FallbackDurationMs = 500

	svc, err := alert.NewAlertService(cfg, beeper, &mockFileSystem{}, nil)
	s.Require().NoError(err)

	err = svc.PlayNotification(context.Background())
	s.NoError(err)

	s.Require().Equal(1, beeper.callCount())
	call := beeper.lastCall()
	s.Equal(float64(440), call.FrequencyHz)
	s.Equal(500, call.DurationMs)
}

// ---------------------------------------------------------------------------
// Cooldown
// ---------------------------------------------------------------------------

func (s *AlertSuite) TestPlayNotificationCooldown() {
	beeper := &mockBeeper{}
	player := &mockAudioPlayer{}
	fsys := &mockFileSystem{entries: entries("ding.mp3")}

	svc, err := alert.NewAlertService(defaultConfig(), beeper, fsys, player)
	s.Require().NoError(err)

	err = svc.PlayNotification(context.Background())
	s.NoError(err)

	// Second call immediately — should be skipped due to cooldown
	err = svc.PlayNotification(context.Background())
	s.NoError(err)

	time.Sleep(100 * time.Millisecond)

	s.Equal(1, player.callCount(), "second call within cooldown should be skipped")
}

func (s *AlertSuite) TestPlayNotificationCooldownExpires() {
	beeper := &mockBeeper{}
	player := &mockAudioPlayer{}
	fsys := &mockFileSystem{entries: entries("ding.mp3")}
	cfg := defaultConfig()
	cfg.AudioCooldownSeconds = 2

	svc, err := alert.NewAlertService(cfg, beeper, fsys, player)
	s.Require().NoError(err)

	err = svc.PlayNotification(context.Background())
	s.NoError(err)

	// Advance past cooldown
	svc.SetNowFunc(func() time.Time {
		return time.Now().Add(3 * time.Second)
	})

	err = svc.PlayNotification(context.Background())
	s.NoError(err)

	time.Sleep(100 * time.Millisecond)

	s.Equal(2, player.callCount(), "second call after cooldown should play")
}

func (s *AlertSuite) TestPlayNotificationConfigurableCooldown() {
	beeper := &mockBeeper{}
	player := &mockAudioPlayer{}
	fsys := &mockFileSystem{entries: entries("ding.mp3")}
	cfg := defaultConfig()
	cfg.AudioCooldownSeconds = 5

	svc, err := alert.NewAlertService(cfg, beeper, fsys, player)
	s.Require().NoError(err)

	err = svc.PlayNotification(context.Background())
	s.NoError(err)

	// 3 seconds later — still within 5s cooldown
	svc.SetNowFunc(func() time.Time {
		return time.Now().Add(3 * time.Second)
	})
	err = svc.PlayNotification(context.Background())
	s.NoError(err)

	time.Sleep(100 * time.Millisecond)
	s.Equal(1, player.callCount(), "3s is within 5s cooldown, should not play again")

	// 6 seconds later — past cooldown
	svc.SetNowFunc(func() time.Time {
		return time.Now().Add(6 * time.Second)
	})
	err = svc.PlayNotification(context.Background())
	s.NoError(err)

	time.Sleep(100 * time.Millisecond)
	s.Equal(2, player.callCount(), "6s is past 5s cooldown, should play again")
}

// ---------------------------------------------------------------------------
// Volume
// ---------------------------------------------------------------------------

func (s *AlertSuite) TestPlayNotificationVolume() {
	player := &mockAudioPlayer{}
	fsys := &mockFileSystem{entries: entries("ding.mp3")}
	cfg := defaultConfig()
	cfg.AudioVolume = 50

	svc, err := alert.NewAlertService(cfg, &mockBeeper{}, fsys, player)
	s.Require().NoError(err)

	err = svc.PlayNotification(context.Background())
	s.NoError(err)

	time.Sleep(100 * time.Millisecond)

	calls := player.getCalls()
	s.Require().Len(calls, 1)
	s.Equal(50, calls[0].Volume)
}

func (s *AlertSuite) TestSetVolume() {
	player := &mockAudioPlayer{}
	fsys := &mockFileSystem{entries: entries("ding.mp3")}
	cfg := defaultConfig()
	cfg.AudioVolume = 50

	svc, err := alert.NewAlertService(cfg, &mockBeeper{}, fsys, player)
	s.Require().NoError(err)

	svc.SetVolume(75)

	err = svc.PlayNotification(context.Background())
	s.NoError(err)

	time.Sleep(100 * time.Millisecond)

	calls := player.getCalls()
	s.Require().Len(calls, 1)
	s.Equal(75, calls[0].Volume)
}

func (s *AlertSuite) TestSetVolumeClamps() {
	player := &mockAudioPlayer{}
	fsys := &mockFileSystem{entries: entries("ding.mp3")}
	cfg := defaultConfig()

	svc, err := alert.NewAlertService(cfg, &mockBeeper{}, fsys, player)
	s.Require().NoError(err)

	// Clamp high
	svc.SetVolume(150)
	svc.SetNowFunc(func() time.Time { return time.Now() })
	err = svc.PlayNotification(context.Background())
	s.NoError(err)

	time.Sleep(100 * time.Millisecond)
	calls := player.getCalls()
	s.Require().Len(calls, 1)
	s.Equal(100, calls[0].Volume, "volume above 100 should clamp to 100")

	// Clamp low
	svc.SetVolume(-10)
	svc.SetNowFunc(func() time.Time { return time.Now().Add(10 * time.Second) })
	err = svc.PlayNotification(context.Background())
	s.NoError(err)

	time.Sleep(100 * time.Millisecond)
	calls = player.getCalls()
	s.Require().Len(calls, 2)
	s.Equal(0, calls[1].Volume, "volume below 0 should clamp to 0")
}

// ---------------------------------------------------------------------------
// Disabled
// ---------------------------------------------------------------------------

func (s *AlertSuite) TestPlayNotificationDisabled() {
	beeper := &mockBeeper{}
	player := &mockAudioPlayer{}
	fsys := &mockFileSystem{entries: entries("ding.mp3")}
	cfg := defaultConfig()
	cfg.AudioEnabled = false

	svc, err := alert.NewAlertService(cfg, beeper, fsys, player)
	s.Require().NoError(err)

	err = svc.PlayNotification(context.Background())
	s.NoError(err)

	time.Sleep(100 * time.Millisecond)

	s.Equal(0, beeper.callCount(), "beeper should not be called when disabled")
	s.Equal(0, player.callCount(), "player should not be called when disabled")
}

// ---------------------------------------------------------------------------
// Error Handling
// ---------------------------------------------------------------------------

func (s *AlertSuite) TestPlayNotificationPlayerErrorFallsBackToBeep() {
	beeper := &mockBeeper{}
	player := &mockAudioPlayer{err: fmt.Errorf("playback failed")}
	fsys := &mockFileSystem{entries: entries("ding.mp3")}

	svc, err := alert.NewAlertService(defaultConfig(), beeper, fsys, player)
	s.Require().NoError(err)

	err = svc.PlayNotification(context.Background())
	s.NoError(err)

	// Wait for async player to fail and fallback to trigger
	time.Sleep(200 * time.Millisecond)

	s.GreaterOrEqual(beeper.callCount(), 1, "beeper should be called as fallback when player errors")
}

func (s *AlertSuite) TestPlayNotificationFsErrorFallsBackToBeep() {
	beeper := &mockBeeper{}
	player := &mockAudioPlayer{}
	fsys := &mockFileSystem{err: fmt.Errorf("permission denied")}

	svc, err := alert.NewAlertService(defaultConfig(), beeper, fsys, player)
	s.Require().NoError(err)

	err = svc.PlayNotification(context.Background())
	s.NoError(err)

	s.GreaterOrEqual(beeper.callCount(), 1, "beeper should be called as fallback when fs errors")
}

// ---------------------------------------------------------------------------
// Path Traversal Protection (G304)
// ---------------------------------------------------------------------------

func (s *AlertSuite) TestSelectAudioFileRejectsPathTraversal() {
	// A malicious filename containing path traversal should be rejected.
	// The directory listing returns a file whose name escapes the audio dir.
	traversalFiles := entries("../../etc/passwd.wav")
	fsys := &mockFileSystem{entries: traversalFiles}
	player := &mockAudioPlayer{}
	beeper := &mockBeeper{}
	cfg := defaultConfig()
	cfg.AudioDir = "/sounds"

	svc, err := alert.NewAlertService(cfg, beeper, fsys, player)
	s.Require().NoError(err)

	err = svc.PlayNotification(context.Background())
	s.NoError(err)

	time.Sleep(100 * time.Millisecond)

	// The traversal file must NOT be played via the audio player.
	// Instead, the service should fall back to beep because no safe files exist.
	s.Equal(0, player.callCount(), "path traversal file should not be played")
	s.GreaterOrEqual(beeper.callCount(), 1, "should fall back to beep when all files are rejected for path traversal")
}

func (s *AlertSuite) TestSelectAudioFilRejectsMixedTraversalAndValid() {
	// When the directory contains both traversal and valid files,
	// only the valid file should ever be selected.
	mixedFiles := entries("../escape.mp3", "safe.mp3", "../../etc/shadow.wav")
	fsys := &mockFileSystem{entries: mixedFiles}
	player := &mockAudioPlayer{}
	beeper := &mockBeeper{}
	cfg := defaultConfig()
	cfg.AudioDir = "/sounds"

	svc, err := alert.NewAlertService(cfg, beeper, fsys, player)
	s.Require().NoError(err)

	// Play multiple times to exercise randomness, advancing past cooldown each time.
	for i := 0; i < 20; i++ {
		svc.SetNowFunc(func() time.Time {
			return time.Now().Add(time.Duration(i*3) * time.Second)
		})
		_ = svc.PlayNotification(context.Background())
	}

	time.Sleep(200 * time.Millisecond)

	calls := player.getCalls()
	for _, call := range calls {
		s.Equal("/sounds/safe.mp3", call.Path,
			"only safe files should be played, got: %s", call.Path)
	}
	s.Greater(len(calls), 0, "safe file should have been played at least once")
}

func (s *AlertSuite) TestBeepPlayerRejectsPathOutsideRoot() {
	// Create a temp directory to act as the audio root.
	audioDir := s.T().TempDir()

	// Place a valid .wav file inside the audio directory.
	// (Content doesn't matter — we're testing path validation, not decoding.)
	wavPath := filepath.Join(audioDir, "valid.wav")
	err := os.WriteFile(wavPath, []byte("RIFF fake wav content"), 0644)
	s.Require().NoError(err)

	// Create a BeepPlayer scoped to audioDir via the new constructor signature.
	player := alert.NewBeepPlayer(audioDir)

	// A file inside the audio directory should NOT fail with a path error.
	// (It will likely fail on decoding since it's not a real wav, but the error
	// should NOT mention path escaping or root violation.)
	errInside := player.PlayFile("valid.wav", 80)
	if errInside != nil {
		s.NotContains(errInside.Error(), "root",
			"file inside audio dir should not be rejected for path reasons")
	}

	// A path that escapes the audio directory MUST be rejected.
	errOutside := player.PlayFile("/etc/passwd", 80)
	s.Require().Error(errOutside, "absolute path outside audio dir must be rejected")
	s.Contains(errOutside.Error(), "root",
		"error should indicate path is outside the root directory")

	// A relative traversal path MUST also be rejected.
	errTraversal := player.PlayFile("../../../etc/passwd", 80)
	s.Require().Error(errTraversal, "traversal path must be rejected")
}

func (s *AlertSuite) TestSelectAudioFileAcceptsNormalPaths() {
	// Normal filenames without traversal components should work fine.
	normalFiles := entries("alert.wav", "notification.mp3")
	fsys := &mockFileSystem{entries: normalFiles}
	player := &mockAudioPlayer{}
	beeper := &mockBeeper{}
	cfg := defaultConfig()
	cfg.AudioDir = "/sounds"

	svc, err := alert.NewAlertService(cfg, beeper, fsys, player)
	s.Require().NoError(err)

	err = svc.PlayNotification(context.Background())
	s.NoError(err)

	time.Sleep(100 * time.Millisecond)

	calls := player.getCalls()
	s.Require().Len(calls, 1, "should play exactly one file")

	validPaths := map[string]bool{
		"/sounds/alert.wav":        true,
		"/sounds/notification.mp3": true,
	}
	s.True(validPaths[calls[0].Path], "played file should be within audio dir, got: %s", calls[0].Path)
}
