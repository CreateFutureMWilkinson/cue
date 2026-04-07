package presenter_test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

// --- Mock VolumeController ---

type mockVolumeController struct {
	mu    sync.Mutex
	calls []int
}

func (m *mockVolumeController) SetVolume(v int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, v)
}

func (m *mockVolumeController) getCalls() []int {
	m.mu.Lock()
	defer m.mu.Unlock()
	dst := make([]int, len(m.calls))
	copy(dst, m.calls)
	return dst
}

// --- Suite ---

type SettingsPresenterSuite struct {
	suite.Suite
}

func TestSettingsPresenter(t *testing.T) {
	suite.Run(t, new(SettingsPresenterSuite))
}

func (s *SettingsPresenterSuite) TestNewSettingsPresenterRequiresVolumeController() {
	_, err := presenter.NewSettingsPresenter(nil, 80, &mockVolumeController{}, 50)
	s.Error(err)
	s.Contains(err.Error(), "volume controller")
}

func (s *SettingsPresenterSuite) TestNewSettingsPresenterSetsInitialVolume() {
	vc := &mockVolumeController{}
	p, err := presenter.NewSettingsPresenter(vc, 80, &mockVolumeController{}, 50)
	s.Require().NoError(err)
	s.Equal(80, p.Volume())
}

func (s *SettingsPresenterSuite) TestSetVolumeUpdatesState() {
	vc := &mockVolumeController{}
	p, err := presenter.NewSettingsPresenter(vc, 80, &mockVolumeController{}, 50)
	s.Require().NoError(err)

	p.SetVolume(50)
	s.Equal(50, p.Volume())
}

func (s *SettingsPresenterSuite) TestSetVolumeDelegatesToController() {
	vc := &mockVolumeController{}
	p, err := presenter.NewSettingsPresenter(vc, 80, &mockVolumeController{}, 50)
	s.Require().NoError(err)

	p.SetVolume(60)
	calls := vc.getCalls()
	s.Require().Len(calls, 1)
	s.Equal(60, calls[0])
}

func (s *SettingsPresenterSuite) TestSetVolumeClampsAbove100() {
	vc := &mockVolumeController{}
	p, err := presenter.NewSettingsPresenter(vc, 50, &mockVolumeController{}, 50)
	s.Require().NoError(err)

	p.SetVolume(150)
	s.Equal(100, p.Volume())

	calls := vc.getCalls()
	s.Require().Len(calls, 1)
	s.Equal(100, calls[0])
}

func (s *SettingsPresenterSuite) TestSetVolumeClampsBelow0() {
	vc := &mockVolumeController{}
	p, err := presenter.NewSettingsPresenter(vc, 50, &mockVolumeController{}, 50)
	s.Require().NoError(err)

	p.SetVolume(-5)
	s.Equal(0, p.Volume())

	calls := vc.getCalls()
	s.Require().Len(calls, 1)
	s.Equal(0, calls[0])
}

func (s *SettingsPresenterSuite) TestTimerVolumeIndependentOfNotificationVolume() {
	notifVC := &mockVolumeController{}
	timerVC := &mockVolumeController{}
	p, err := presenter.NewSettingsPresenter(notifVC, 80, timerVC, 65)
	s.Require().NoError(err)

	// Initial timer volume is set from constructor
	s.Equal(65, p.TimerVolume())

	// SetTimerVolume updates state and delegates to timer controller
	p.SetTimerVolume(40)
	s.Equal(40, p.TimerVolume())

	timerCalls := timerVC.getCalls()
	s.Require().Len(timerCalls, 1)
	s.Equal(40, timerCalls[0])

	// Notification volume is unaffected
	s.Equal(80, p.Volume())
	notifCalls := notifVC.getCalls()
	s.Empty(notifCalls)

	// Timer volume clamps above 100
	p.SetTimerVolume(150)
	s.Equal(100, p.TimerVolume())

	// Timer volume clamps below 0
	p.SetTimerVolume(-10)
	s.Equal(0, p.TimerVolume())
}
