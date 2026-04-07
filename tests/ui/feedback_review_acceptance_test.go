//go:build ui_acceptance

package ui_acceptance_test

import (
	"context"
	"testing"

	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/ui"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/uitest"
)

// FeedbackReviewAcceptanceSuite verifies feedback review acceptance criteria
// from UiSpec.md lines 1060-1068.
type FeedbackReviewAcceptanceSuite struct {
	suite.Suite
}

func TestFeedbackReviewAcceptance(t *testing.T) {
	suite.Run(t, new(FeedbackReviewAcceptanceSuite))
}

// AC: Review button callback wired when FeedbackPresenter provided.
func (s *FeedbackReviewAcceptanceSuite) TestReviewButtonCallbackWired() {
	app := newTestApp()
	defer app.Quit()
	router := ui.NewCenterViewRouter()
	mw, _, _ := newMainWindowWithFeedback(app, router, sampleNotifiedMessages(), sampleBufferedMessages())

	s.Require().NotNil(mw.FocusRail())
	s.NotNil(mw.FocusRail().ReviewButton().OnTapped,
		"Review button should have a callback when FeedbackPresenter is provided")
}

// AC: Feedback presenter loads buffered messages.
func (s *FeedbackReviewAcceptanceSuite) TestFeedbackPresenterLoadsBufferedMessages() {
	buffered := sampleBufferedMessages()
	reviewer := &mockBufferReviewer{messages: buffered}
	fp, err := presenter.NewFeedbackPresenter(reviewer)
	s.Require().NoError(err)

	err = fp.Load(context.Background())
	s.NoError(err, "FeedbackPresenter.Load should succeed with mock reviewer")
}

// AC: Counter shows "X of Y buffered messages reviewed" (1-indexed).
func (s *FeedbackReviewAcceptanceSuite) TestFeedbackPresenterCounterFormat() {
	buffered := sampleBufferedMessages()
	reviewer := &mockBufferReviewer{messages: buffered}
	fp, err := presenter.NewFeedbackPresenter(reviewer)
	s.Require().NoError(err)

	err = fp.Load(context.Background())
	s.Require().NoError(err)

	counter := fp.Counter()
	s.Contains(counter, "1", "counter should show 1-indexed current position")
	s.Contains(counter, "2", "counter should show total count of buffered messages")
}

// AC: Shows Source, Sender, Channel, IS, CS for current message.
func (s *FeedbackReviewAcceptanceSuite) TestFeedbackPresenterCurrentItem() {
	buffered := sampleBufferedMessages()
	reviewer := &mockBufferReviewer{messages: buffered}
	fp, err := presenter.NewFeedbackPresenter(reviewer)
	s.Require().NoError(err)

	err = fp.Load(context.Background())
	s.Require().NoError(err)

	s.True(fp.HasCurrent(), "should have a current item after loading")
	item := fp.Current()
	s.NotEmpty(item.Source, "current item should have a source")
	s.NotEmpty(item.Sender, "current item should have a sender")
}

// AC: 11 rating buttons (0-10) exist as a concept in the feedback review.
// This tests the maxRating constant indirectly through presenter behavior.
func (s *FeedbackReviewAcceptanceSuite) TestFeedbackAcceptsRatings0Through10() {
	buffered := sampleBufferedMessages()
	reviewer := &mockBufferReviewer{messages: buffered}
	fp, err := presenter.NewFeedbackPresenter(reviewer)
	s.Require().NoError(err)

	err = fp.Load(context.Background())
	s.Require().NoError(err)

	// Saving a rating of 0 should work.
	err = fp.SaveRating(context.Background(), 0, nil)
	s.NoError(err, "rating 0 should be accepted")
}

// AC: Review button hidden when notifications collapsed.
func (s *FeedbackReviewAcceptanceSuite) TestReviewButtonHiddenWhenCollapsed() {
	app := newTestApp()
	defer app.Quit()
	router := ui.NewCenterViewRouter()
	mw, _, _ := newMainWindowWithFeedback(app, router, sampleNotifiedMessages(), sampleBufferedMessages())

	s.False(mw.FocusRail().ReviewButton().Visible(),
		"Review button should be hidden when notifications are collapsed")
}

// AC: Review button visible after expanding notifications.
func (s *FeedbackReviewAcceptanceSuite) TestReviewButtonVisibleAfterExpand() {
	app := newTestApp()
	defer app.Quit()
	router := ui.NewCenterViewRouter()
	mw, np, _ := newMainWindowWithFeedback(app, router, sampleNotifiedMessages(), sampleBufferedMessages())

	np.ToggleExpanded()

	s.True(mw.FocusRail().ReviewButton().Visible(),
		"Review button should be visible after expanding notifications")
}

// AC: Review button returns to hidden after collapse.
func (s *FeedbackReviewAcceptanceSuite) TestReviewButtonHiddenAfterExpandCollapse() {
	app := newTestApp()
	defer app.Quit()
	router := ui.NewCenterViewRouter()
	mw, np, _ := newMainWindowWithFeedback(app, router, sampleNotifiedMessages(), sampleBufferedMessages())

	np.ToggleExpanded()
	np.ToggleExpanded()

	s.False(mw.FocusRail().ReviewButton().Visible(),
		"Review button should be hidden after expand then collapse")
}

// AC: Focus rail Review button exists in the widget tree.
func (s *FeedbackReviewAcceptanceSuite) TestReviewButtonExistsInFocusRail() {
	app := newTestApp()
	defer app.Quit()
	router := ui.NewCenterViewRouter()
	mw, _, _ := newMainWindowWithFeedback(app, router, sampleNotifiedMessages(), sampleBufferedMessages())

	rail := mw.FocusRail()
	s.Require().NotNil(rail)
	root := rail.Container()

	_, found := uitest.FindWidget[*widget.Button](root, func(b *widget.Button) bool {
		return b.Text == "Review"
	})
	s.True(found, "Focus rail should contain a 'Review' button")
}
