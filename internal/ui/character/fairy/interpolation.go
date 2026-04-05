package fairy

// EaseInOut computes Hermite smoothstep interpolation.
// The result is 0 at t=0, 0.5 at t=0.5, and 1 at t=1.
func EaseInOut(t float64) float64 {
	return t * t * (3.0 - 2.0*t)
}
