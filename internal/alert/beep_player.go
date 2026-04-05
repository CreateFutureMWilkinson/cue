package alert

import (
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/effects"
	"github.com/gopxl/beep/v2/mp3"
	"github.com/gopxl/beep/v2/speaker"
	"github.com/gopxl/beep/v2/vorbis"
	"github.com/gopxl/beep/v2/wav"
)

// wrapMp3Decode adapts mp3.Decode to the common decoder signature.
func wrapMp3Decode(rc io.ReadCloser) (beep.StreamSeekCloser, beep.Format, error) {
	return mp3.Decode(rc)
}

// wrapWavDecode adapts wav.Decode (which takes io.Reader) to the common decoder signature.
func wrapWavDecode(rc io.ReadCloser) (beep.StreamSeekCloser, beep.Format, error) {
	return wav.Decode(rc)
}

// wrapVorbisDecode adapts vorbis.Decode to the common decoder signature.
func wrapVorbisDecode(rc io.ReadCloser) (beep.StreamSeekCloser, beep.Format, error) {
	return vorbis.Decode(rc)
}

// DecodeMp3Fn is the function used to decode MP3 files. Override in tests.
var DecodeMp3Fn = wrapMp3Decode

// DecodeWavFn is the function used to decode WAV files. Override in tests.
var DecodeWavFn = wrapWavDecode

// DecodeVorbisFn is the function used to decode OGG/Vorbis files. Override in tests.
var DecodeVorbisFn = wrapVorbisDecode

// SpeakerInitFn is the function used to initialize the speaker. Override in tests.
var SpeakerInitFn = speaker.Init

// SpeakerPlayFn is the function used to play audio streams. Override in tests.
var SpeakerPlayFn = func(s ...beep.Streamer) { speaker.Play(s...) }

// BeepPlayer implements AudioPlayer using the gopxl/beep library.
type BeepPlayer struct {
	audioDir   string
	initOnce   sync.Once
	sampleRate beep.SampleRate
	initErr    error
}

// NewBeepPlayer creates a new BeepPlayer.
//
// If audioDir is provided, file access is scoped to that directory using os.OpenRoot (Go 1.24+).
// If no audioDir is provided, files are opened directly using os.Open.
func NewBeepPlayer(audioDir ...string) *BeepPlayer {
	p := &BeepPlayer{}
	if len(audioDir) > 0 {
		p.audioDir = audioDir[0]
	}
	return p
}

// MapVolume converts a 0-100 integer volume to a beep volume value and silent flag.
//
// Volume mapping:
//   - 0 or negative: silent (muted)
//   - 100 or higher: 0.0 (full volume, no attenuation)
//   - 1-99: logarithmic scaling using log2(volume/100)
//
// The logarithmic formula provides more natural volume perception, where:
//   - volume=50 → log2(0.5) ≈ -1.0 dB (half perceived loudness)
//   - volume=25 → log2(0.25) = -2.0 dB (quarter perceived loudness)
//
// This matches human hearing characteristics better than linear scaling.
func MapVolume(volume int) (beepVol float64, silent bool) {
	if volume <= 0 {
		return 0, true
	}
	if volume >= 100 {
		return 0.0, false
	}
	return math.Log2(float64(volume) / 100.0), false
}

// PlayFile plays an audio file at the given volume (0-100).
func (p *BeepPlayer) PlayFile(path string, volume int) error {
	streamer, format, err := p.decodeAudioFile(path)
	if err != nil {
		return fmt.Errorf("decoding audio file %q: %w", path, err)
	}
	defer streamer.Close()

	if err := p.ensureSpeakerInitialized(format); err != nil {
		return fmt.Errorf("ensuring speaker initialized: %w", err)
	}

	volumeStreamer := p.createVolumeStreamer(streamer, format, volume)
	SpeakerPlayFn(volumeStreamer)

	return nil
}

// decodeAudioFile opens and decodes an audio file, returning the streamer and format.
func (p *BeepPlayer) decodeAudioFile(path string) (beep.StreamSeekCloser, beep.Format, error) {
	f, err := p.openAudioFile(path)
	if err != nil {
		return nil, beep.Format{}, err
	}
	defer f.Close()

	ext := strings.ToLower(filepath.Ext(path))
	var streamer beep.StreamSeekCloser
	var format beep.Format

	switch ext {
	case ".mp3":
		streamer, format, err = DecodeMp3Fn(f)
	case ".wav":
		streamer, format, err = DecodeWavFn(f)
	case ".ogg":
		streamer, format, err = DecodeVorbisFn(f)
	default:
		return nil, beep.Format{}, fmt.Errorf("unsupported audio format: %s", ext)
	}

	if err != nil {
		return nil, beep.Format{}, fmt.Errorf("decoding %s file: %w", ext, err)
	}

	return streamer, format, nil
}

// openAudioFile opens an audio file, using scoped access if audioDir is configured.
func (p *BeepPlayer) openAudioFile(path string) (*os.File, error) {
	if p.audioDir != "" {
		return p.openWithRootDir(path)
	}
	return p.openDirect(path)
}

// openWithRootDir opens a file using os.OpenRoot for scoped access.
func (p *BeepPlayer) openWithRootDir(path string) (*os.File, error) {
	root, err := os.OpenRoot(p.audioDir)
	if err != nil {
		return nil, fmt.Errorf("opening root directory: %w", err)
	}
	defer root.Close()

	f, err := root.Open(path)
	if err != nil {
		return nil, fmt.Errorf("path outside root directory: %w", err)
	}
	return f, nil
}

// openDirect opens a file directly using os.Open.
func (p *BeepPlayer) openDirect(path string) (*os.File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	return f, nil
}

// ensureSpeakerInitialized initializes the speaker using sync.Once if not already done.
func (p *BeepPlayer) ensureSpeakerInitialized(format beep.Format) error {
	p.initOnce.Do(func() {
		p.sampleRate = format.SampleRate
		p.initErr = SpeakerInitFn(format.SampleRate, format.SampleRate.N(time.Second/10))
	})
	return p.initErr
}

// createVolumeStreamer creates a volume-controlled streamer with optional resampling.
func (p *BeepPlayer) createVolumeStreamer(streamer beep.StreamSeekCloser, format beep.Format, volume int) beep.Streamer {
	var s beep.Streamer = streamer
	if format.SampleRate != p.sampleRate {
		s = beep.Resample(4, format.SampleRate, p.sampleRate, streamer)
	}

	beepVol, silent := MapVolume(volume)
	return &effects.Volume{
		Streamer: s,
		Base:     2,
		Volume:   beepVol,
		Silent:   silent,
	}
}
