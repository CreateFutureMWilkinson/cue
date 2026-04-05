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
	if volume < 0 {
		volume = 0
	}
	if volume > 100 {
		volume = 100
	}
	p.volume = volume
	p.vc.SetVolume(volume)
}
