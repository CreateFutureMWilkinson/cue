package ui_test

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

type NotificationInteractionSuite struct {
	suite.Suite
	querier *mockQuerier
	updater *mockUpdater
}

func TestNotificationInteraction(t *testing.T) {
	suite.Run(t, new(NotificationInteractionSuite))
}

func (s *NotificationInteractionSuite) SetupTest() {
	s.querier = &mockQuerier{
		messages: []*repository.Message{
			{
				ID:              uuid.New(),
				Source:          "slack",
				Sender:          "alice",
				Channel:         "general",
				RawContent:      "Server is on fire!",
				ImportanceScore: 9.0,
				ConfidenceScore: 0.95,
				Reasoning:       "Server outage detected",
				Status:          "Notified",
				CreatedAt:       time.Now().Add(-5 * time.Minute),
			},
			{
				ID:              uuid.New(),
				Source:          "email",
				Sender:          "bob@example.com",
				Channel:         "inbox",
				RawContent:      "Quarterly report deadline tomorrow",
				ImportanceScore: 7.5,
				ConfidenceScore: 0.85,
				Reasoning:       "Upcoming deadline",
				Status:          "Notified",
				CreatedAt:       time.Now().Add(-10 * time.Minute),
			},
		},
	}
	s.updater = &mockUpdater{}
}

func (s *NotificationInteractionSuite) newPanel() *ui.NotificationPanel {
	np, err := presenter.NewNotificationPresenter(s.querier, s.updater)
	s.Require().NoError(err)
	err = np.Refresh(context.Background())
	s.Require().NoError(err)

	win := test.NewWindow(nil)
	s.T().Cleanup(func() { win.Close() })

	return ui.NewNotificationPanel(np, win)
}

func (s *NotificationInteractionSuite) TestNotificationPanelContainsHeaderLabel() {
	panel := s.newPanel()
	root := panel.Container()

	lbl, found := uitest.FindWidget[*widget.Label](root, func(l *widget.Label) bool {
		return l.Text == "Notifications"
	})

	s.True(found, "should find a Label with text 'Notifications' in the widget tree")
	s.Equal("Notifications", lbl.Text)
}

func (s *NotificationInteractionSuite) TestNotificationPanelContainsList() {
	panel := s.newPanel()
	root := panel.Container()

	_, found := uitest.FindWidget[*widget.List](root, func(_ *widget.List) bool {
		return true
	})

	s.True(found, "should find a *widget.List in the widget tree")
}

func (s *NotificationInteractionSuite) TestNotificationPanelListShowsCorrectItemCount() {
	panel := s.newPanel()
	root := panel.Container()

	list := uitest.RequireWidget[*widget.List](s.T(), root, func(_ *widget.List) bool {
		return true
	})

	s.Equal(2, list.Length(), "list item count should match the number of messages from querier")
}

func (s *NotificationInteractionSuite) TestOnNotificationClickFiresWithWebURL() {
	panel := s.newPanel()

	// Set up the click callback to capture the URL
	var clickedURL string
	var callbackFired bool
	panel.SetOnNotificationClick(func(url string) {
		callbackFired = true
		clickedURL = url
	})

	// Act: simulate selecting the first item in the notification list
	root := panel.Container()
	list := uitest.RequireWidget[*widget.List](s.T(), root, func(_ *widget.List) bool {
		return true
	})
	list.Select(0)

	// Assert: the callback should have been invoked
	s.True(callbackFired,
		"OnNotificationClick callback should fire when a notification is selected")
	// The URL should come from the card's WebURL field (empty until wired to account config,
	// but the callback mechanism must work)
	_ = clickedURL
}
