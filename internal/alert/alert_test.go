package alert_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/alert"
	"github.com/stretchr/testify/suite"
)

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

type beepCall struct {
	FrequencyHz float64
	DurationMs  int
}

type mockBeeper struct {
	calls []beepCall
	err   error
}

func (m *mockBeeper) Beep(frequencyHz float64, durationMs int) error {
	m.calls = append(m.calls, beepCall{FrequencyHz: frequencyHz, DurationMs: durationMs})
	return m.err
}

// ---------------------------------------------------------------------------
// Suite
// ---------------------------------------------------------------------------

type AlertSuite struct {
	suite.Suite
}

func TestAlert(t *testing.T) {
	suite.Run(t, new(AlertSuite))
}

// ---------------------------------------------------------------------------
// Constructor Validation
// ---------------------------------------------------------------------------

func (s *AlertSuite) TestNewAlertServiceRequiresBeeper() {
	svc, err := alert.NewAlertService(alert.AlertConfig{AudioEnabled: true}, nil)
	s.Error(err)
	s.Nil(svc)
	s.Contains(err.Error(), "beeper")
}

func (s *AlertSuite) TestNewAlertServiceAcceptsValidDeps() {
	svc, err := alert.NewAlertService(alert.AlertConfig{AudioEnabled: true}, &mockBeeper{})
	s.NoError(err)
	s.NotNil(svc)
}

// ---------------------------------------------------------------------------
// PlayNotification
// ---------------------------------------------------------------------------

func (s *AlertSuite) TestPlayNotificationWhenEnabled() {
	beeper := &mockBeeper{}
	svc, err := alert.NewAlertService(alert.AlertConfig{AudioEnabled: true}, beeper)
	s.Require().NoError(err)

	err = svc.PlayNotification(context.Background())
	s.NoError(err)

	s.Require().Len(beeper.calls, 1)
	s.Equal(float64(1000), beeper.calls[0].FrequencyHz)
	s.Equal(200, beeper.calls[0].DurationMs)
}

func (s *AlertSuite) TestPlayNotificationWhenDisabled() {
	beeper := &mockBeeper{}
	svc, err := alert.NewAlertService(alert.AlertConfig{AudioEnabled: false}, beeper)
	s.Require().NoError(err)

	err = svc.PlayNotification(context.Background())
	s.NoError(err)

	s.Len(beeper.calls, 0)
}

func (s *AlertSuite) TestPlayNotificationCooldown() {
	beeper := &mockBeeper{}
	svc, err := alert.NewAlertService(alert.AlertConfig{AudioEnabled: true}, beeper)
	s.Require().NoError(err)

	err = svc.PlayNotification(context.Background())
	s.NoError(err)

	// Second call immediately — should be skipped due to 2s cooldown
	err = svc.PlayNotification(context.Background())
	s.NoError(err)

	s.Len(beeper.calls, 1)
}

func (s *AlertSuite) TestPlayNotificationCooldownExpires() {
	beeper := &mockBeeper{}
	svc, err := alert.NewAlertService(alert.AlertConfig{AudioEnabled: true}, beeper)
	s.Require().NoError(err)

	err = svc.PlayNotification(context.Background())
	s.NoError(err)

	// Simulate time passing beyond cooldown
	svc.SetNowFunc(func() time.Time {
		return time.Now().Add(3 * time.Second)
	})

	err = svc.PlayNotification(context.Background())
	s.NoError(err)

	s.Len(beeper.calls, 2)
}

// ---------------------------------------------------------------------------
// PlayStartup
// ---------------------------------------------------------------------------

func (s *AlertSuite) TestPlayStartupWhenEnabled() {
	beeper := &mockBeeper{}
	svc, err := alert.NewAlertService(alert.AlertConfig{AudioEnabled: true}, beeper)
	s.Require().NoError(err)

	err = svc.PlayStartup(context.Background())
	s.NoError(err)

	s.Require().Len(beeper.calls, 1)
	s.Equal(float64(600), beeper.calls[0].FrequencyHz)
	s.Equal(150, beeper.calls[0].DurationMs)
}

func (s *AlertSuite) TestPlayStartupWhenDisabled() {
	beeper := &mockBeeper{}
	svc, err := alert.NewAlertService(alert.AlertConfig{AudioEnabled: false}, beeper)
	s.Require().NoError(err)

	err = svc.PlayStartup(context.Background())
	s.NoError(err)

	s.Len(beeper.calls, 0)
}

// ---------------------------------------------------------------------------
// PlayShutdown
// ---------------------------------------------------------------------------

func (s *AlertSuite) TestPlayShutdownWhenEnabled() {
	beeper := &mockBeeper{}
	svc, err := alert.NewAlertService(alert.AlertConfig{AudioEnabled: true}, beeper)
	s.Require().NoError(err)

	err = svc.PlayShutdown(context.Background())
	s.NoError(err)

	s.Require().Len(beeper.calls, 1)
	s.Equal(float64(400), beeper.calls[0].FrequencyHz)
	s.Equal(300, beeper.calls[0].DurationMs)
}

func (s *AlertSuite) TestPlayShutdownWhenDisabled() {
	beeper := &mockBeeper{}
	svc, err := alert.NewAlertService(alert.AlertConfig{AudioEnabled: false}, beeper)
	s.Require().NoError(err)

	err = svc.PlayShutdown(context.Background())
	s.NoError(err)

	s.Len(beeper.calls, 0)
}

// ---------------------------------------------------------------------------
// Error Handling
// ---------------------------------------------------------------------------

func (s *AlertSuite) TestPlayNotificationBeeperError() {
	beeper := &mockBeeper{err: fmt.Errorf("audio device unavailable")}
	svc, err := alert.NewAlertService(alert.AlertConfig{AudioEnabled: true}, beeper)
	s.Require().NoError(err)

	err = svc.PlayNotification(context.Background())
	s.Error(err)
	s.Contains(err.Error(), "audio device unavailable")
}
