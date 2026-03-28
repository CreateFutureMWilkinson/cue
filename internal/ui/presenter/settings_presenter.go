package presenter

import "fmt"

// SettingsPresenter manages application settings state.
type SettingsPresenter struct {
	vc     VolumeController
	volume int
}

// NewSettingsPresenter creates a new SettingsPresenter with the given volume
// controller and initial volume.
func NewSettingsPresenter(vc VolumeController, initialVolume int) (*SettingsPresenter, error) {
	if vc == nil {
		return nil, fmt.Errorf("volume controller must not be nil")
	}
	return &SettingsPresenter{
		vc:     vc,
		volume: initialVolume,
	}, nil
}

// Volume returns the current volume level.
func (p *SettingsPresenter) Volume() int {
	return p.volume
}

// SetVolume updates the volume, clamping to 0-100, and delegates to the
// volume controller.
func (p *SettingsPresenter) SetVolume(volume int) {
	p.volume = clampVolume(volume)
	p.vc.SetVolume(p.volume)
}

// clampVolume ensures the volume is within the valid range of 0-100.
func clampVolume(volume int) int {
	if volume < 0 {
		return 0
	}
	if volume > 100 {
		return 100
	}
	return volume
}
