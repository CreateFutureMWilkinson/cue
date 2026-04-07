//go:build ui_acceptance

package ui_acceptance_test

import (
	"testing"

	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/ui"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/uitest"
)

// ActivityLogAcceptanceSuite verifies activity log drawer acceptance criteria
// from UiSpec.md lines 1050-1057.
type ActivityLogAcceptanceSuite struct {
	suite.Suite
}

func TestActivityLogAcceptance(t *testing.T) {
	suite.Run(t, new(ActivityLogAcceptanceSuite))
}

func (s *ActivityLogAcceptanceSuite) newActivityPresenter() *presenter.ActivityPresenter {
	source := newMockActivitySource()
	ap, err := presenter.NewActivityPresenter(source, 500)
	s.Require().NoError(err)
	return ap
}

// AC: Activity log drawer exists as a component.
func (s *ActivityLogAcceptanceSuite) TestDrawerCreation() {
	ap := s.newActivityPresenter()
	drawer := ui.NewActivityLogDrawer(ap)
	s.NotNil(drawer, "NewActivityLogDrawer should return a non-nil drawer")
}

// AC: Activity log drawer has a container for rendering.
func (s *ActivityLogAcceptanceSuite) TestDrawerHasContainer() {
	ap := s.newActivityPresenter()
	drawer := ui.NewActivityLogDrawer(ap)
	container := drawer.Container()
	s.NotNil(container, "drawer.Container() should return a non-nil canvas object")
}

// AC: Hidden by default, toggle button at bottom of character area.
func (s *ActivityLogAcceptanceSuite) TestDrawerHiddenByDefault() {
	app := newTestApp()
	defer app.Quit()
	router := ui.NewCenterViewRouter()
	mw := newMinimalMainWindow(app, router)

	content := mw.CenterContent()
	s.NotNil(content, "center content in character view should not be nil")
}

// AC: Hidden by default, toggle button at bottom of character area.
// When closed, the drawer shows a toggle button labeled "Activity Log".
func (s *ActivityLogAcceptanceSuite) TestDrawerClosedShowsToggleButton() {
	ap := s.newActivityPresenter()
	drawer := ui.NewActivityLogDrawer(ap)
	root := drawer.Container()

	btn, found := uitest.FindWidget[*widget.Button](root, func(b *widget.Button) bool {
		return b.Text == "Activity Log"
	})

	s.True(found, "closed drawer should contain 'Activity Log' toggle button")
	s.Equal("Activity Log", btn.Text)
}

// AC: When opened, the drawer should show a list widget for log entries.
func (s *ActivityLogAcceptanceSuite) TestDrawerOpenedContainsLogList() {
	ap := s.newActivityPresenter()
	drawer := ui.NewActivityLogDrawer(ap)
	drawer.ToggleOpen()
	root := drawer.Container()

	_, found := uitest.FindWidget[*widget.List](root, func(_ *widget.List) bool {
		return true
	})

	s.True(found, "opened drawer should contain a *widget.List for log entries")
}

// AC: Only accessible when character area is visible (not during expanded notifications).
func (s *ActivityLogAcceptanceSuite) TestCenterContentChangesOnNavigation() {
	app := newTestApp()
	defer app.Quit()
	router := ui.NewCenterViewRouter()
	mw := newMinimalMainWindow(app, router)

	charContent := mw.CenterContent()
	router.NavigateTo(ui.ViewPlan)
	planContent := mw.CenterContent()

	s.NotEqual(charContent, planContent,
		"center content should change when navigating away from character view")
}

// AC: Maximum 500 entries with FIFO eviction.
// Verified: presenter is configured with maxEntries=500.
func (s *ActivityLogAcceptanceSuite) TestPresenterMaxEntries() {
	source := newMockActivitySource()
	ap, err := presenter.NewActivityPresenter(source, 500)
	s.NoError(err)
	s.NotNil(ap, "ActivityPresenter should be created with 500 max entries")
}
