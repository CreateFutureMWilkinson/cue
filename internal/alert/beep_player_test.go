package alert_test

import (
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/CreateFutureMWilkinson/cue/internal/alert"
	"github.com/gopxl/beep/v2"
	"github.com/stretchr/testify/suite"
)

// ---------------------------------------------------------------------------
// Mock streamer (implements beep.StreamSeekCloser)
// ---------------------------------------------------------------------------

type mockStreamer struct{}

func (m *mockStreamer) Stream(samples [][2]float64) (int, bool) { return 0, false }
func (m *mockStreamer) Err() error                              { return nil }
func (m *mockStreamer) Len() int                                { return 0 }
func (m *mockStreamer) Position() int                           { return 0 }
func (m *mockStreamer) Seek(p int) error                        { return nil }
func (m *mockStreamer) Close() error                            { return nil }

// ---------------------------------------------------------------------------
// Suite
// ---------------------------------------------------------------------------

type BeepPlayerSuite struct {
	suite.Suite
}

func TestBeepPlayer(t *testing.T) {
	suite.Run(t, new(BeepPlayerSuite))
}

// ---------------------------------------------------------------------------
// Constructor tests
// ---------------------------------------------------------------------------

func (s *BeepPlayerSuite) TestNewBeepPlayerReturnsNonNil() {
	player := alert.NewBeepPlayer()
	s.NotNil(player)
}

func (s *BeepPlayerSuite) TestNewBeepPlayerImplementsAudioPlayer() {
	var _ alert.AudioPlayer = (*alert.BeepPlayer)(nil)
}

// ---------------------------------------------------------------------------
// Volume mapping tests
// ---------------------------------------------------------------------------

func (s *BeepPlayerSuite) TestMapVolumeZeroIsSilent() {
	_, silent := alert.MapVolume(0)
	s.True(silent)
}

func (s *BeepPlayerSuite) TestMapVolumeFull() {
	vol, silent := alert.MapVolume(100)
	s.False(silent)
	s.InDelta(0.0, vol, 0.001)
}

func (s *BeepPlayerSuite) TestMapVolumeHalf() {
	vol, silent := alert.MapVolume(50)
	s.False(silent)
	s.InDelta(math.Log2(0.5), vol, 0.001) // ≈ -1.0
}

func (s *BeepPlayerSuite) TestMapVolumeClampsNegative() {
	_, silent := alert.MapVolume(-5)
	s.True(silent)
}

func (s *BeepPlayerSuite) TestMapVolumeClampsOver100() {
	vol, silent := alert.MapVolume(150)
	s.False(silent)
	s.InDelta(0.0, vol, 0.001)
}

// ---------------------------------------------------------------------------
// Format detection / decoder dispatch tests
// ---------------------------------------------------------------------------

func (s *BeepPlayerSuite) TestPlayFileDispatchesMp3Decoder() {
	tmpDir := s.T().TempDir()
	tmpFile := filepath.Join(tmpDir, "test.mp3")
	s.Require().NoError(os.WriteFile(tmpFile, []byte("fake"), 0644))

	called := false

	origMp3 := alert.DecodeMp3Fn
	s.T().Cleanup(func() { alert.DecodeMp3Fn = origMp3 })
	alert.DecodeMp3Fn = func(rc io.ReadCloser) (beep.StreamSeekCloser, beep.Format, error) {
		called = true
		rc.Close()
		return nil, beep.Format{}, errors.New("mock: stop after decode dispatch")
	}

	player := alert.NewBeepPlayer()
	_ = player.PlayFile(tmpFile, 50)

	s.True(called, "expected mp3 decoder to be called")
}

func (s *BeepPlayerSuite) TestPlayFileDispatchesWavDecoder() {
	tmpDir := s.T().TempDir()
	tmpFile := filepath.Join(tmpDir, "test.wav")
	s.Require().NoError(os.WriteFile(tmpFile, []byte("fake"), 0644))

	called := false

	origWav := alert.DecodeWavFn
	s.T().Cleanup(func() { alert.DecodeWavFn = origWav })
	alert.DecodeWavFn = func(rc io.ReadCloser) (beep.StreamSeekCloser, beep.Format, error) {
		called = true
		rc.Close()
		return nil, beep.Format{}, errors.New("mock: stop after decode dispatch")
	}

	player := alert.NewBeepPlayer()
	_ = player.PlayFile(tmpFile, 50)

	s.True(called, "expected wav decoder to be called")
}

func (s *BeepPlayerSuite) TestPlayFileDispatchesOggDecoder() {
	tmpDir := s.T().TempDir()
	tmpFile := filepath.Join(tmpDir, "test.ogg")
	s.Require().NoError(os.WriteFile(tmpFile, []byte("fake"), 0644))

	called := false

	origVorbis := alert.DecodeVorbisFn
	s.T().Cleanup(func() { alert.DecodeVorbisFn = origVorbis })
	alert.DecodeVorbisFn = func(rc io.ReadCloser) (beep.StreamSeekCloser, beep.Format, error) {
		called = true
		rc.Close()
		return nil, beep.Format{}, errors.New("mock: stop after decode dispatch")
	}

	player := alert.NewBeepPlayer()
	_ = player.PlayFile(tmpFile, 50)

	s.True(called, "expected vorbis decoder to be called")
}

func (s *BeepPlayerSuite) TestPlayFileUnsupportedExtension() {
	tmpDir := s.T().TempDir()
	tmpFile := filepath.Join(tmpDir, "test.flac")
	s.Require().NoError(os.WriteFile(tmpFile, []byte("fake"), 0644))

	player := alert.NewBeepPlayer()
	err := player.PlayFile(tmpFile, 50)

	s.Error(err)
	s.Contains(err.Error(), "unsupported")
}

// ---------------------------------------------------------------------------
// Error handling tests
// ---------------------------------------------------------------------------

func (s *BeepPlayerSuite) TestPlayFileNonexistentFile() {
	player := alert.NewBeepPlayer()
	err := player.PlayFile("/nonexistent/path/audio.mp3", 50)
	s.Error(err)
}

func (s *BeepPlayerSuite) TestPlayFileDecoderError() {
	tmpDir := s.T().TempDir()
	tmpFile := filepath.Join(tmpDir, "test.mp3")
	s.Require().NoError(os.WriteFile(tmpFile, []byte("fake"), 0644))

	origMp3 := alert.DecodeMp3Fn
	s.T().Cleanup(func() { alert.DecodeMp3Fn = origMp3 })
	alert.DecodeMp3Fn = func(rc io.ReadCloser) (beep.StreamSeekCloser, beep.Format, error) {
		rc.Close()
		return nil, beep.Format{}, errors.New("corrupt file")
	}

	player := alert.NewBeepPlayer()
	err := player.PlayFile(tmpFile, 50)

	s.Error(err)
	s.Contains(err.Error(), "corrupt file")
}

// ---------------------------------------------------------------------------
// Speaker init-once test
// ---------------------------------------------------------------------------

func (s *BeepPlayerSuite) TestSpeakerInitCalledOnce() {
	tmpDir := s.T().TempDir()
	file1 := filepath.Join(tmpDir, "a.mp3")
	file2 := filepath.Join(tmpDir, "b.mp3")
	s.Require().NoError(os.WriteFile(file1, []byte("fake"), 0644))
	s.Require().NoError(os.WriteFile(file2, []byte("fake"), 0644))

	var initCount atomic.Int32
	mock := &mockStreamer{}
	format := beep.Format{SampleRate: 44100, NumChannels: 2, Precision: 2}

	// Save and restore all overridden function variables
	origMp3 := alert.DecodeMp3Fn
	origInit := alert.SpeakerInitFn
	origPlay := alert.SpeakerPlayFn
	s.T().Cleanup(func() {
		alert.DecodeMp3Fn = origMp3
		alert.SpeakerInitFn = origInit
		alert.SpeakerPlayFn = origPlay
	})

	alert.DecodeMp3Fn = func(rc io.ReadCloser) (beep.StreamSeekCloser, beep.Format, error) {
		rc.Close()
		return mock, format, nil
	}

	alert.SpeakerInitFn = func(sr beep.SampleRate, bufferSize int) error {
		initCount.Add(1)
		return nil
	}

	alert.SpeakerPlayFn = func(streamers ...beep.Streamer) {
		// Immediately drain — simulate instant playback completion.
		// The real speaker.Play is async; the implementation must block
		// until done via a signal/callback streamer. For this test we
		// just need it to not hang, so we invoke the callback streamer
		// if present.
		for _, st := range streamers {
			if cb, ok := st.(beep.StreamSeekCloser); ok {
				_ = cb
			}
		}
	}

	player := alert.NewBeepPlayer()

	// First call — should trigger speaker init
	_ = player.PlayFile(file1, 50)
	// Second call — should NOT trigger speaker init again
	_ = player.PlayFile(file2, 50)

	s.Equal(int32(1), initCount.Load(), "speaker.Init should be called exactly once")
}
