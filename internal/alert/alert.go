package alert

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gen2brain/beeep"
)

// AlertConfig holds configuration for the alert service.
type AlertConfig struct {
	AudioEnabled bool
}

// Beeper abstracts audio beep functionality.
type Beeper interface {
	Beep(frequencyHz float64, durationMs int) error
}

// beeepBeeper wraps the beeep package for production use.
type beeepBeeper struct{}

func (b *beeepBeeper) Beep(frequencyHz float64, durationMs int) error {
	return beeep.Beep(frequencyHz, durationMs)
}

// AlertService manages audio alerts with cooldown support.
type AlertService struct {
	cfg       AlertConfig
	beeper    Beeper
	lastAlert time.Time
	mu        sync.Mutex
	now       func() time.Time
}

// NewAlertService creates a new AlertService, validating dependencies.
func NewAlertService(cfg AlertConfig, beeper Beeper) (*AlertService, error) {
	if beeper == nil {
		return nil, fmt.Errorf("beeper is required")
	}
	return &AlertService{
		cfg:    cfg,
		beeper: beeper,
		now:    time.Now,
	}, nil
}

// SetNowFunc allows tests to inject a clock function for cooldown testing.
func (a *AlertService) SetNowFunc(fn func() time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.now = fn
}

// PlayNotification plays a notification alert with 2-second cooldown.
func (a *AlertService) PlayNotification(ctx context.Context) error {
	if !a.cfg.AudioEnabled {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	now := a.now()
	if !a.lastAlert.IsZero() && now.Sub(a.lastAlert) < 2*time.Second {
		return nil
	}
	a.lastAlert = now

	return a.beeper.Beep(1000, 200)
}

// PlayStartup plays a startup chime.
func (a *AlertService) PlayStartup(ctx context.Context) error {
	if !a.cfg.AudioEnabled {
		return nil
	}
	return a.beeper.Beep(600, 150)
}

// PlayShutdown plays a shutdown tone.
func (a *AlertService) PlayShutdown(ctx context.Context) error {
	if !a.cfg.AudioEnabled {
		return nil
	}
	return a.beeper.Beep(400, 300)
}
