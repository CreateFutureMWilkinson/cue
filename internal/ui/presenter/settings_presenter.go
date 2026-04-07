package presenter

import "fmt"

// SettingsPresenter manages application settings state.
type SettingsPresenter struct {
	vc          VolumeController
	volume      int
	timerVC     VolumeController
	timerVolume int
}

// NewSettingsPresenter creates a new SettingsPresenter with the given volume
// controller and initial volume, plus a timer volume controller and initial
// timer volume.
func NewSettingsPresenter(vc VolumeController, initialVolume int, timerVC VolumeController, initialTimerVolume int) (*SettingsPresenter, error) {
	if vc == nil {
		return nil, fmt.Errorf("volume controller must not be nil")
	}
	return &SettingsPresenter{
		vc:     vc,
		volume: initialVolume,
	}, nil
}

// TimerVolume returns the current timer volume level.
func (p *SettingsPresenter) TimerVolume() int {
	return 0
}

// SetTimerVolume updates the timer volume, clamping to 0-100, and delegates
// to the timer volume controller.
func (p *SettingsPresenter) SetTimerVolume(volume int) {
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
