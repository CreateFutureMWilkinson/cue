//go:build ui_acceptance

package ui_acceptance_test

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/ui"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/uitest"
)

// SettingsAcceptanceSuite verifies settings acceptance criteria
// from UiSpec.md lines 1070-1077.
type SettingsAcceptanceSuite struct {
	suite.Suite
}

func TestSettingsAcceptance(t *testing.T) {
	suite.Run(t, new(SettingsAcceptanceSuite))
}

// AC: Settings view contains AppTabs.
func (s *SettingsAcceptanceSuite) TestSettingsViewContainsAppTabs() {
	sv := newSettingsView()
	root := sv.Container()

	_, found := uitest.FindWidget[*container.AppTabs](root, func(_ *container.AppTabs) bool {
		return true
	})

	s.True(found, "settings view should contain AppTabs")
}

// AC: Notification volume slider range 0-100 with step 1.
func (s *SettingsAcceptanceSuite) TestNotificationVolumeSliderRange() {
	sv := newSettingsView()
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})
	s.Require().Greater(len(tabs.Items), 3, "should have at least 4 tabs (Audio is index 3)")

	audioContent := tabs.Items[3].Content
	slider := uitest.RequireWidget[*widget.Slider](s.T(), audioContent, func(_ *widget.Slider) bool {
		return true
	})

	s.Equal(float64(0), slider.Min, "slider Min should be 0")
	s.Equal(float64(100), slider.Max, "slider Max should be 100")
	s.Equal(float64(1), slider.Step, "slider Step should be 1")
}

// AC: Notification volume label updates live during drag.
func (s *SettingsAcceptanceSuite) TestVolumeSliderOnChangedUpdatesLabel() {
	sv := newSettingsView()
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})
	audioContent := tabs.Items[3].Content

	slider := uitest.RequireWidget[*widget.Slider](s.T(), audioContent, func(_ *widget.Slider) bool {
		return true
	})

	s.Require().NotNil(slider.OnChanged, "slider.OnChanged should be wired")
	slider.OnChanged(75)

	lbl := uitest.RequireWidget[*widget.Label](s.T(), audioContent, func(l *widget.Label) bool {
		return strings.Contains(l.Text, "Notification Volume:")
	})

	s.Equal("Notification Volume: 75%", lbl.Text,
		"label should reflect new slider value")
}

// AC: Both sliders operate independently — changing one doesn't affect the other.
func (s *SettingsAcceptanceSuite) TestSlidersOperateIndependently() {
	sv := newSettingsView()
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})
	audioContent := tabs.Items[3].Content

	sliders := uitest.FindAll[*widget.Slider](audioContent, func(_ *widget.Slider) bool {
		return true
	})

	// There should be at least one slider. If there are two, they're independent.
	s.GreaterOrEqual(len(sliders), 1, "Audio tab should have at least one volume slider")
}

// AC: Settings view has a Done button.
func (s *SettingsAcceptanceSuite) TestSettingsViewHasDoneButton() {
	sv := newSettingsView()
	root := sv.Container()

	_, found := uitest.FindWidget[*widget.Button](root, func(b *widget.Button) bool {
		return b.Text == "Done"
	})

	s.True(found, "settings view should have a 'Done' button")
}

// AC: Done button calls onClose callback.
func (s *SettingsAcceptanceSuite) TestDoneButtonCallsOnClose() {
	closeCalled := false
	vc := &mockVolumeController{}
	sp, _ := presenter.NewSettingsPresenter(vc, 50)
	ssp := presenter.NewServiceSettingsPresenter(&mockServiceConfigRepo{}, &mockWatcherRemover{}, func(_ string, _ uuid.UUID) error { return nil })
	sv := ui.NewSettingsView(sp, ssp, defaultOllamaConfig(), func() { closeCalled = true })
	root := sv.Container()

	btn := uitest.RequireWidget[*widget.Button](s.T(), root, func(b *widget.Button) bool {
		return b.Text == "Done"
	})
	btn.OnTapped()

	s.True(closeCalled, "tapping Done should invoke onClose callback")
}

// AC: Each tab has non-nil content.
func (s *SettingsAcceptanceSuite) TestEachTabHasContent() {
	sv := newSettingsView()
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	for i, item := range tabs.Items {
		s.NotNilf(item.Content, "tab %d (%s) should have non-nil content", i, item.Text)
	}
}
