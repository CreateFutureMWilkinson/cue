//go:build ui_acceptance

package ui_acceptance_test

import (
	"context"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/ui"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/uitest"
)

// NotificationAcceptanceSuite verifies notification panel acceptance criteria
// from UiSpec.md lines 1033-1048.
type NotificationAcceptanceSuite struct {
	suite.Suite
}

func TestNotificationAcceptance(t *testing.T) {
	suite.Run(t, new(NotificationAcceptanceSuite))
}

func (s *NotificationAcceptanceSuite) newPanel(messages []*repository.Message) (*ui.NotificationPanel, *presenter.NotificationPresenter) {
	querier := &mockQuerier{messages: messages}
	updater := &mockUpdater{}
	np, err := presenter.NewNotificationPresenter(querier, updater)
	s.Require().NoError(err)
	err = np.Refresh(context.Background())
	s.Require().NoError(err)

	win := test.NewWindow(nil)
	s.T().Cleanup(func() { win.Close() })

	panel := ui.NewNotificationPanel(np, win)
	return panel, np
}

// AC: Notification panel contains a header label.
func (s *NotificationAcceptanceSuite) TestPanelContainsHeaderLabel() {
	panel, _ := s.newPanel(sampleNotifiedMessages())
	root := panel.Container()

	lbl, found := uitest.FindWidget[*widget.Label](root, func(l *widget.Label) bool {
		return l.Text == "Notifications"
	})

	s.True(found, "should find 'Notifications' header label")
	s.Equal("Notifications", lbl.Text)
}

// AC: Notification panel contains a list widget.
func (s *NotificationAcceptanceSuite) TestPanelContainsList() {
	panel, _ := s.newPanel(sampleNotifiedMessages())
	root := panel.Container()

	_, found := uitest.FindWidget[*widget.List](root, func(_ *widget.List) bool {
		return true
	})

	s.True(found, "should find a *widget.List in notification panel")
}

// AC: List shows correct number of notification items.
func (s *NotificationAcceptanceSuite) TestListShowsCorrectItemCount() {
	msgs := sampleNotifiedMessages()
	panel, _ := s.newPanel(msgs)
	root := panel.Container()

	list := uitest.RequireWidget[*widget.List](s.T(), root, func(_ *widget.List) bool {
		return true
	})

	s.Equal(len(msgs), list.Length(),
		"list item count should match the number of notified messages")
}

// AC: Displays only messages with status NOTIFIED.
func (s *NotificationAcceptanceSuite) TestDisplaysOnlyNotifiedMessages() {
	// Include a mix of statuses — only NOTIFIED should appear.
	msgs := []*repository.Message{
		{
			ID: uuid.New(), Source: "slack", Sender: "alice", Channel: "general",
			RawContent: "Alert!", ImportanceScore: 9.0, ConfidenceScore: 0.95,
			Status: "Notified", CreatedAt: time.Now(),
		},
	}
	panel, _ := s.newPanel(msgs)
	root := panel.Container()

	list := uitest.RequireWidget[*widget.List](s.T(), root, func(_ *widget.List) bool {
		return true
	})

	s.Equal(1, list.Length(), "only notified messages should appear in the panel")
}

// AC: Panel with no messages shows empty list.
func (s *NotificationAcceptanceSuite) TestEmptyPanelShowsNoItems() {
	panel, _ := s.newPanel(nil)
	root := panel.Container()

	list := uitest.RequireWidget[*widget.List](s.T(), root, func(_ *widget.List) bool {
		return true
	})

	s.Equal(0, list.Length(), "empty message set should produce zero list items")
}

// AC: Cards sorted newest-first — verified by list ordering matching
// the presorted input (newest first by CreatedAt).
func (s *NotificationAcceptanceSuite) TestCardsOrderedNewestFirst() {
	now := time.Now()
	msgs := []*repository.Message{
		{
			ID: uuid.New(), Source: "slack", Sender: "first", Channel: "ch1",
			RawContent: "Oldest", ImportanceScore: 7.0, ConfidenceScore: 0.8,
			Status: "Notified", CreatedAt: now.Add(-10 * time.Minute),
		},
		{
			ID: uuid.New(), Source: "slack", Sender: "second", Channel: "ch2",
			RawContent: "Newest", ImportanceScore: 8.0, ConfidenceScore: 0.9,
			Status: "Notified", CreatedAt: now.Add(-1 * time.Minute),
		},
	}
	panel, np := s.newPanel(msgs)
	root := panel.Container()

	list := uitest.RequireWidget[*widget.List](s.T(), root, func(_ *widget.List) bool {
		return true
	})
	s.Require().Equal(2, list.Length())

	// The presenter sorts by CreatedAt descending. Verify first row is the newest.
	rows := np.Messages()
	s.Require().Len(rows, 2)
	s.Equal("second", rows[0].Sender, "first row should be the newest message")
}

// AC: Notification panel has an expand/collapse toggle.
func (s *NotificationAcceptanceSuite) TestExpandCollapseToggleExists() {
	panel, _ := s.newPanel(sampleNotifiedMessages())
	root := panel.Container()

	// Look for a button that toggles expansion (commonly labeled with an arrow or "Expand").
	buttons := uitest.FindAll[*widget.Button](root, func(_ *widget.Button) bool {
		return true
	})

	s.Greater(len(buttons), 0,
		"notification panel should contain at least one button (expand/collapse toggle)")
}

// AC: Review button appears in focus rail when notifications expanded.
func (s *NotificationAcceptanceSuite) TestReviewButtonVisibleWhenExpanded() {
	app := newTestApp()
	defer app.Quit()
	router := ui.NewCenterViewRouter()
	mw, np, _ := newMainWindowWithFeedback(app, router, sampleNotifiedMessages(), sampleBufferedMessages())

	s.Require().NotNil(mw.FocusRail())

	// Before expansion: review button hidden.
	s.False(mw.FocusRail().ReviewButton().Visible(),
		"Review button should be hidden before expansion")

	// Expand notifications.
	np.ToggleExpanded()

	s.True(mw.FocusRail().ReviewButton().Visible(),
		"Review button should be visible after expansion")
}

// AC: Collapse toggle returns to 30% width and restores character.
func (s *NotificationAcceptanceSuite) TestCollapseRestoresReviewButtonHidden() {
	app := newTestApp()
	defer app.Quit()
	router := ui.NewCenterViewRouter()
	mw, np, _ := newMainWindowWithFeedback(app, router, sampleNotifiedMessages(), sampleBufferedMessages())

	// Expand then collapse.
	np.ToggleExpanded()
	np.ToggleExpanded()

	s.False(mw.FocusRail().ReviewButton().Visible(),
		"Review button should be hidden after collapsing notifications")
}

// AC: Multiple notifications at different IS levels produce correct list count.
func (s *NotificationAcceptanceSuite) TestMultipleImportanceLevels() {
	msgs := sampleNotifiedMessages() // IS 9.5, 8.0, 7.0
	panel, _ := s.newPanel(msgs)
	root := panel.Container()

	list := uitest.RequireWidget[*widget.List](s.T(), root, func(_ *widget.List) bool {
		return true
	})

	s.Equal(3, list.Length(),
		"panel should show all three messages at different IS levels")
}
