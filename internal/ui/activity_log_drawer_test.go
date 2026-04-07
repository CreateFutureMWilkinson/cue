package ui_test

import (
	"testing"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/ui"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/uitest"
)

// --- mock ActivitySource ---

type mockActivitySource struct {
	ch chan presenter.ActivityEvent
}

func newMockActivitySource() *mockActivitySource {
	return &mockActivitySource{ch: make(chan presenter.ActivityEvent, 10)}
}

func (m *mockActivitySource) Events() <-chan presenter.ActivityEvent {
	return m.ch
}

// --- Suite ---

type ActivityLogDrawerSuite struct {
	suite.Suite
	source *mockActivitySource
}

func TestActivityLogDrawer(t *testing.T) {
	suite.Run(t, new(ActivityLogDrawerSuite))
}

func (s *ActivityLogDrawerSuite) SetupTest() {
	s.source = newMockActivitySource()
}

func (s *ActivityLogDrawerSuite) newPresenter() *presenter.ActivityPresenter {
	ap, err := presenter.NewActivityPresenter(s.source, 500)
	s.Require().NoError(err)
	return ap
}

func (s *ActivityLogDrawerSuite) TestNewActivityLogDrawerNotNil() {
	ap := s.newPresenter()

	drawer := ui.NewActivityLogDrawer(ap)

	s.NotNil(drawer, "NewActivityLogDrawer should return a non-nil drawer")
}

func (s *ActivityLogDrawerSuite) TestDrawerDefaultHidden() {
	ap := s.newPresenter()

	drawer := ui.NewActivityLogDrawer(ap)

	s.False(drawer.IsOpen(), "drawer should be hidden (closed) by default")
}

func (s *ActivityLogDrawerSuite) TestDrawerToggleOpen() {
	ap := s.newPresenter()

	drawer := ui.NewActivityLogDrawer(ap)
	drawer.ToggleOpen()

	s.True(drawer.IsOpen(), "drawer should be open after ToggleOpen()")
}

func (s *ActivityLogDrawerSuite) TestDrawerToggleClose() {
	ap := s.newPresenter()

	drawer := ui.NewActivityLogDrawer(ap)
	drawer.ToggleOpen() // open
	drawer.ToggleOpen() // close

	s.False(drawer.IsOpen(), "drawer should be closed after toggling twice")
}

func (s *ActivityLogDrawerSuite) TestDrawerContainerNotNil() {
	ap := s.newPresenter()

	drawer := ui.NewActivityLogDrawer(ap)

	s.NotNil(drawer.Container(), "Container() should return a non-nil canvas object")
}

func (s *ActivityLogDrawerSuite) TestDrawerContainerWithCharacterWidget() {
	ap := s.newPresenter()

	drawer := ui.NewActivityLogDrawer(ap)
	characterWidget := widget.NewLabel("Fairy placeholder")

	container := drawer.ContainerWithCharacter(characterWidget)

	s.NotNil(container, "ContainerWithCharacter should return a non-nil container")
}

func (s *ActivityLogDrawerSuite) TestDrawerContainerWithNilCharacterWidget() {
	ap := s.newPresenter()

	drawer := ui.NewActivityLogDrawer(ap)

	container := drawer.ContainerWithCharacter(nil)

	s.NotNil(container, "ContainerWithCharacter(nil) should still return a non-nil container")
}

func (s *ActivityLogDrawerSuite) TestContainerWithCharacterUsesStackNotSplit() {
	ap := s.newPresenter()

	drawer := ui.NewActivityLogDrawer(ap)
	drawer.ToggleOpen()
	characterWidget := widget.NewLabel("Fairy placeholder")

	result := drawer.ContainerWithCharacter(characterWidget)

	_, found := uitest.FindWidget[*container.Split](result, func(_ *container.Split) bool {
		return true
	})
	s.False(found, "ContainerWithCharacter should use a Stack layout, not a Split")
}
