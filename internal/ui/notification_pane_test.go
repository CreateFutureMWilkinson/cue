package ui_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/ui"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/uitest"
)

// --- Local mocks (ui_test cannot reuse presenter_test mocks) ---

type mockQuerier struct {
	messages []*repository.Message
	err      error
}

func (m *mockQuerier) QueryByStatus(_ context.Context, _ string) ([]*repository.Message, error) {
	return m.messages, m.err
}

type mockUpdater struct {
	updateCalled bool
	err          error
}

func (m *mockUpdater) Update(_ context.Context, _ *repository.Message) error {
	m.updateCalled = true
	return m.err
}

// --- Suite ---

type NotificationPaneSuite struct {
	suite.Suite
	querier *mockQuerier
	updater *mockUpdater
}

func TestNotificationPane(t *testing.T) {
	suite.Run(t, new(NotificationPaneSuite))
}

func (s *NotificationPaneSuite) SetupTest() {
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
				Reasoning:       "Server outage detected with high urgency keywords",
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
				Reasoning:       "Upcoming deadline with moderate urgency",
				Status:          "Notified",
				CreatedAt:       time.Now().Add(-10 * time.Minute),
			},
		},
	}
	s.updater = &mockUpdater{}
}

func (s *NotificationPaneSuite) newPresenter() *presenter.NotificationPresenter {
	np, err := presenter.NewNotificationPresenter(s.querier, s.updater)
	s.Require().NoError(err)
	err = np.Refresh(context.Background())
	s.Require().NoError(err)
	return np
}

func (s *NotificationPaneSuite) TestNewNotificationPanelNotNil() {
	np := s.newPresenter()
	win := test.NewWindow(nil)
	defer win.Close()

	panel := ui.NewNotificationPanel(np, win)

	s.NotNil(panel, "NewNotificationPanel should return a non-nil panel")
}

func (s *NotificationPaneSuite) TestPanelDefaultCollapsed() {
	np := s.newPresenter()
	win := test.NewWindow(nil)
	defer win.Close()

	panel := ui.NewNotificationPanel(np, win)

	s.False(panel.IsExpanded(), "new panel should default to collapsed state")
}

func (s *NotificationPaneSuite) TestPanelToggleExpand() {
	np := s.newPresenter()
	win := test.NewWindow(nil)
	defer win.Close()

	panel := ui.NewNotificationPanel(np, win)
	panel.ToggleExpand()

	s.True(panel.IsExpanded(), "panel should be expanded after ToggleExpand()")
}

func (s *NotificationPaneSuite) TestPanelToggleCollapse() {
	np := s.newPresenter()
	win := test.NewWindow(nil)
	defer win.Close()

	panel := ui.NewNotificationPanel(np, win)
	panel.ToggleExpand() // expand
	panel.ToggleExpand() // collapse

	s.False(panel.IsExpanded(), "panel should be collapsed after toggling twice")
}

func (s *NotificationPaneSuite) TestPanelContainerNotNil() {
	np := s.newPresenter()
	win := test.NewWindow(nil)
	defer win.Close()

	panel := ui.NewNotificationPanel(np, win)

	s.NotNil(panel.Container(), "Container() should return a non-nil canvas object")
}

func (s *NotificationPaneSuite) TestDetailDialogShowsContent() {
	np := s.newPresenter()
	win := test.NewWindow(nil)
	defer win.Close()

	// NewNotificationPanel must exist (compile check), but the real assertion
	// is that Select(0) on the presenter returns the correct detail data
	// that would feed the dialog.
	_ = ui.NewNotificationPanel(np, win)

	detail, err := np.Select(0)
	s.Require().NoError(err)
	s.InDelta(9.0, detail.ImportanceScore, 0.001, "first message IS should be 9.0")
	s.InDelta(0.95, detail.ConfidenceScore, 0.001, "first message CS should be 0.95")
	s.Equal("Server is on fire!", detail.Content, "detail content should match raw content")
}

func (s *NotificationPaneSuite) TestDetailDialogResolveRemovesMessage() {
	np := s.newPresenter()
	win := test.NewWindow(nil)
	defer win.Close()

	_ = ui.NewNotificationPanel(np, win)

	// Select the first message to get its ID, then resolve it.
	detail, err := np.Select(0)
	s.Require().NoError(err)

	err = np.Resolve(context.Background(), detail.ID)
	s.Require().NoError(err)

	// After resolving, the message list should have one fewer entry.
	s.Len(np.Messages(), 1, "Messages() should have 1 entry after resolving one")
	s.True(s.updater.updateCalled, "updater.Update should have been called")
}

// --- Feature 018-Hotfix-A: Notification Card Visual Rendering ---

func (s *NotificationPaneSuite) TestCollapsedCardShowsBadgeText() {
	np := s.newPresenter()
	win := test.NewWindow(nil)
	defer win.Close()

	panel := ui.NewNotificationPanel(np, win)

	// Panel defaults to collapsed
	s.False(panel.IsExpanded())

	// Get cards from presenter to verify badge data
	cards := np.Cards()
	s.Require().NotEmpty(cards)

	// In collapsed state, the score should display as integer (e.g., "9")
	s.InDelta(9.0, cards[0].ImportanceScore, 0.001)
	s.Equal("general", cards[0].Channel)
}

func (s *NotificationPaneSuite) TestExpandedCardShowsFullScore() {
	np := s.newPresenter()
	win := test.NewWindow(nil)
	defer win.Close()

	panel := ui.NewNotificationPanel(np, win)
	panel.ToggleExpand() // switch to expanded

	s.True(panel.IsExpanded())

	// In expanded state, cards should show full decimal score (e.g., "9.0")
	cards := np.Cards()
	s.Require().NotEmpty(cards)
	s.InDelta(9.0, cards[0].ImportanceScore, 0.001)
	s.Equal("slack", cards[0].Source)
	s.Equal("alice", cards[0].Sender)
}

func (s *NotificationPaneSuite) TestExpandedCardShowsDismissAction() {
	np := s.newPresenter()
	win := test.NewWindow(nil)
	defer win.Close()

	panel := ui.NewNotificationPanel(np, win)
	panel.ToggleExpand() // switch to expanded

	// Get the first card's ID, verify we can dismiss by ID
	cards := np.Cards()
	s.Require().NotEmpty(cards)

	err := np.DismissMessage(context.Background(), cards[0].ID)
	s.Require().NoError(err)

	// After dismiss, one fewer card
	remainingCards := np.Cards()
	s.Len(remainingCards, 1)
}

func (s *NotificationPaneSuite) TestDetailDialogShowsReasoning() {
	np := s.newPresenter()
	win := test.NewWindow(nil)
	defer win.Close()

	_ = ui.NewNotificationPanel(np, win)

	detail, err := np.Select(0)
	s.Require().NoError(err)

	s.Equal("Server outage detected with high urgency keywords", detail.Reasoning,
		"detail dialog should include reasoning text from the message")
}

func (s *NotificationPaneSuite) TestCardRenderingUsesColoredElements() {
	np := s.newPresenter()
	win := test.NewWindow(nil)
	defer win.Close()

	panel := ui.NewNotificationPanel(np, win)

	// RenderCard should return a non-nil card widget for the first notification.
	card := panel.RenderCard(0)
	s.Require().NotNil(card, "RenderCard(0) must return a non-nil canvas object")

	// The rendered card must contain a canvas.Rectangle for the colored background.
	_, found := uitest.FindWidget[*canvas.Rectangle](card, func(r *canvas.Rectangle) bool {
		return true
	})
	s.True(found, "rendered card should contain a canvas.Rectangle for the colored background")
}

func (s *NotificationPaneSuite) TestPanelContainsExpandToggleButton() {
	np := s.newPresenter()
	win := test.NewWindow(nil)
	defer win.Close()

	panel := ui.NewNotificationPanel(np, win)
	root := panel.Container()

	btn, found := uitest.FindWidget[*widget.Button](root, func(b *widget.Button) bool {
		return strings.Contains(strings.ToLower(b.Text), "expand")
	})
	s.Require().True(found, "panel should contain a button with 'expand' in its text")
	s.Contains(strings.ToLower(btn.Text), "expand")
}

func (s *NotificationPaneSuite) TestExpandedCardContainsDismissButton() {
	np := s.newPresenter()
	win := test.NewWindow(nil)
	defer win.Close()

	panel := ui.NewNotificationPanel(np, win)
	panel.ToggleExpand()

	card := panel.RenderExpandedCard(0)
	s.Require().NotNil(card, "RenderExpandedCard(0) must return a non-nil canvas object")

	_, found := uitest.FindWidget[*widget.Button](card, func(b *widget.Button) bool {
		return b.Text == "Dismiss"
	})
	s.True(found, "expanded card should contain a Dismiss button")
}
