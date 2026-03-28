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
	initOnce   sync.Once
	sampleRate beep.SampleRate
	initErr    error
}

// NewBeepPlayer creates a new BeepPlayer.
func NewBeepPlayer() *BeepPlayer {
	return &BeepPlayer{}
}

// MapVolume converts a 0-100 integer volume to a beep volume value and silent flag.
// Volume 0 or negative returns silent. Volume >= 100 returns 0.0 (full volume).
// Values in between are mapped via log2(volume/100).
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
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening audio file: %w", err)
	}

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
		f.Close()
		return fmt.Errorf("unsupported audio format: %s", ext)
	}
	if err != nil {
		return fmt.Errorf("decoding %s: %w", ext, err)
	}
	defer streamer.Close()

	p.initOnce.Do(func() {
		p.sampleRate = format.SampleRate
		p.initErr = SpeakerInitFn(format.SampleRate, format.SampleRate.N(time.Second/10))
	})
	if p.initErr != nil {
		return fmt.Errorf("initializing speaker: %w", p.initErr)
	}

	var s beep.Streamer = streamer
	if format.SampleRate != p.sampleRate {
		s = beep.Resample(4, format.SampleRate, p.sampleRate, streamer)
	}

	beepVol, silent := MapVolume(volume)
	volumeS := &effects.Volume{
		Streamer: s,
		Base:     2,
		Volume:   beepVol,
		Silent:   silent,
	}

	SpeakerPlayFn(volumeS)

	return nil
}
