package alert

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gen2brain/beeep"
)

// Audio constants for notification sounds.
const (
	notificationFreqHz     = 1000.0
	notificationDurationMs = 200
	startupFreqHz          = 600.0
	startupDurationMs      = 150
	shutdownFreqHz         = 400.0
	shutdownDurationMs     = 300
	notificationCooldown   = 2 * time.Second
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

// NewBeeepBeeper creates a new production beeper using the beeep package.
func NewBeeepBeeper() Beeper {
	return &beeepBeeper{}
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
	if !a.lastAlert.IsZero() && now.Sub(a.lastAlert) < notificationCooldown {
		return nil
	}
	a.lastAlert = now

	return a.beeper.Beep(notificationFreqHz, notificationDurationMs)
}

// PlayStartup plays a startup chime.
func (a *AlertService) PlayStartup(ctx context.Context) error {
	if !a.cfg.AudioEnabled {
		return nil
	}
	return a.beeper.Beep(startupFreqHz, startupDurationMs)
}

// PlayShutdown plays a shutdown tone.
func (a *AlertService) PlayShutdown(ctx context.Context) error {
	if !a.cfg.AudioEnabled {
		return nil
	}
	return a.beeper.Beep(shutdownFreqHz, shutdownDurationMs)
}
