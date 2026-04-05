# Feature 13: gopxl/beep AudioPlayer Integration

**Phase:** Phase-1-Feature-13 (amendment)
**Status:** Done
**Package:** `internal/alert/`

---

## Overview

Implements the `AudioPlayer` interface using `gopxl/beep/v2` for real MP3/WAV/OGG file playback. Previously the player was `nil` (always falling back to `beeep` tones). Now audio files from the user's configured `audio_dir` are decoded and played through the system audio output.

This is the sole CGO dependency in the project — `gopxl/beep/v2` requires `ebitengine/oto/v3` which uses CGO for ALSA bindings on Linux.

## Design Decisions

### Lazy Speaker Initialization

`speaker.Init(sampleRate, bufferSize)` must be called once before any playback, but requires knowing the sample rate. Rather than hardcoding a rate or requiring it in the constructor, the first `PlayFile` call initializes the speaker with that file's native sample rate via `sync.Once`.

### Automatic Resampling

After the first file sets the speaker rate, subsequent files with different native sample rates are automatically resampled using `beep.Resample(quality=4, nativeRate, speakerRate, streamer)`. Quality 4 balances speed and fidelity for notification sounds.

### Logarithmic Volume Mapping

The `effects.Volume` struct uses `Base^Volume` as the amplitude multiplier. With `Base=2`:

| User Volume | Formula | Beep Volume | Multiplier |
|---|---|---|---|
| 0 | silent | n/a | 0 (muted) |
| 50 | log2(0.5) | -1.0 | 0.5x |
| 100 | log2(1.0) | 0.0 | 1.0x (full) |

This provides perceptually natural volume scaling — the user's linear 0-100 slider maps to a logarithmic amplitude curve.

### Fire-and-Forget Playback

`PlayFile` dispatches audio to `speaker.Play` and returns immediately. Since `PlayFile` is already called in a goroutine by `AlertService`, this avoids double-blocking. Errors during decoding or speaker init are returned synchronously.

### Injectable Test Seams

Five exported package-level function variables allow tests to mock the audio subsystem without playing real audio:

- `DecodeMp3Fn`, `DecodeWavFn`, `DecodeVorbisFn` — decoder dispatch
- `SpeakerInitFn` — speaker initialization
- `SpeakerPlayFn` — audio output

Tests override these and restore via `s.T().Cleanup()`.

## API

```go
// Constructor (no args, no error)
func NewBeepPlayer() *BeepPlayer

// Implements AudioPlayer interface
func (p *BeepPlayer) PlayFile(path string, volume int) error

// Exported for testability
func MapVolume(volume int) (beepVol float64, silent bool)
```

## Error Handling

| Error | Behavior |
|---|---|
| File not found | Return error (caller falls back to beeep) |
| Unsupported extension | Return error with "unsupported" |
| Decoder failure | Return wrapped error |
| Speaker init failure | Return wrapped error |
| Playback failure | Silent (fire-and-forget after dispatch) |

## Test Coverage

| Area | Tests |
|---|---|
| Constructor | 2 |
| Volume mapping | 5 |
| Format dispatch | 4 |
| Error handling | 2 |
| Speaker init-once | 1 |
| **Total** | **14** |

## TDD Agent Stats

| TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | Test Designer | 80s | 25,749 | 58c7dd2 |
| GREEN | Implementer | 186s | 31,918 | 8e90df1 |
| REFACTOR | Refactorer | 58s | 22,723 | 79fae83 |
