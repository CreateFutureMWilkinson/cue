package fairy

import (
	"image/color"
	"math"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
)

const (
	// Idle position coordinates.
	IdleOriginX = 0.5
	IdleOriginY = 1.0
)

var (
	// IdleBodyColor is the idle state body color (#00FF00).
	IdleBodyColor = color.RGBA{R: 0x00, G: 0xFF, B: 0x00, A: 0xFF}

	// initialFairyColor is the default body color (#00FF00).
	initialFairyColor = IdleBodyColor

	// State colors for the fairy character indicator.
	colorIdle         = color.RGBA{R: 200, G: 200, B: 255, A: 255} // Light blue
	colorStarting     = color.RGBA{R: 255, G: 255, B: 200, A: 255} // Light yellow
	colorWorking      = color.RGBA{R: 200, G: 255, B: 200, A: 255} // Light green
	colorNotifying    = color.RGBA{R: 255, G: 200, B: 100, A: 255} // Orange
	colorError        = color.RGBA{R: 255, G: 100, B: 100, A: 255} // Light red
	colorShuttingDown = color.RGBA{R: 150, G: 150, B: 150, A: 255} // Gray

	// glowBaseAlphas stores the graduated base alpha values for each glow layer.
	glowBaseAlphas = [fairyGlowLayerCount]uint8{128, 112, 96, 80, 64, 48, 32, 16}
)

// clamp01 clamps a value to the range [0.0, 1.0].
func clamp01(v float64) float64 {
	if v < 0.0 {
		return 0.0
	}
	if v > 1.0 {
		return 1.0
	}
	return v
}

// lerpColor linearly interpolates between two RGBA colors.
func lerpColor(a, b color.RGBA, t float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(a.R) + t*(float64(b.R)-float64(a.R))),
		G: uint8(float64(a.G) + t*(float64(b.G)-float64(a.G))),
		B: uint8(float64(a.B) + t*(float64(b.B)-float64(a.B))),
		A: uint8(float64(a.A) + t*(float64(b.A)-float64(a.A))),
	}
}

// glowIntensity computes the glow intensity at time t using a sinusoidal
// breathing pattern. The result oscillates between min and max intensities
// with the specified period.
func glowIntensity(t, period, min, max float64) float64 {
	phase := 2 * math.Pi * t / period
	sinWave := math.Sin(phase)
	normalizedSin := (sinWave + 1.0) / 2.0
	return min + (max-min)*normalizedSin
}

func stateColor(s character.CharacterState) color.Color {
	switch s {
	case character.StateIdle:
		return colorIdle
	case character.StateStarting:
		return colorStarting
	case character.StateWorking:
		return colorWorking
	case character.StateNotifying:
		return colorNotifying
	case character.StateError:
		return colorError
	case character.StateShuttingDown:
		return colorShuttingDown
	default:
		return colorIdle
	}
}
