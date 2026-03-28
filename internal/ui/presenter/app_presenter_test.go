package presenter_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

// --- Mock Alerter ---

type mockAlerter struct {
	mu             sync.Mutex
	startupCalled  bool
	shutdownCalled bool
	startupErr     error
	shutdownErr    error
}

func (m *mockAlerter) PlayStartup(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startupCalled = true
	return m.startupErr
}

func (m *mockAlerter) PlayShutdown(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shutdownCalled = true
	return m.shutdownErr
}

// --- Helper: minimal mock deps for constructing real sub-presenters ---

type stubMessageQuerier struct {
	messages []*repository.Message
}

func (s *stubMessageQuerier) QueryByStatus(_ context.Context, _ string) ([]*repository.Message, error) {
	return s.messages, nil
}

type stubMessageUpdater struct{}

func (s *stubMessageUpdater) Update(_ context.Context, _ *repository.Message) error {
	return nil
}

type stubActivitySource struct {
	ch chan presenter.ActivityEvent
}

func newStubActivitySource() *stubActivitySource {
	return &stubActivitySource{ch: make(chan presenter.ActivityEvent, 100)}
}

func (s *stubActivitySource) Events() <-chan presenter.ActivityEvent {
	return s.ch
}

type stubBufferReviewer struct{}

func (s *stubBufferReviewer) GetBufferedMessages(_ context.Context) ([]*repository.Message, error) {
	return nil, nil
}

func (s *stubBufferReviewer) CountBuffered(_ context.Context) (int, error) {
	return 0, nil
}

func (s *stubBufferReviewer) SaveRating(_ context.Context, _ uuid.UUID, _ int, _ *string) error {
	return nil
}

func (s *stubBufferReviewer) DeleteMessage(_ context.Context, _ uuid.UUID) error {
	return nil
}

// --- Suite ---

type AppPresenterSuite struct {
	suite.Suite
	notifQuerier *stubMessageQuerier
	actSource    *stubActivitySource
	notification *presenter.NotificationPresenter
	activity     *presenter.ActivityPresenter
	feedback     *presenter.FeedbackPresenter
	alerter      *mockAlerter
}

func TestAppPresenter(t *testing.T) {
	suite.Run(t, new(AppPresenterSuite))
}

func (s *AppPresenterSuite) SetupTest() {
	s.notifQuerier = &stubMessageQuerier{}
	s.actSource = newStubActivitySource()
	s.alerter = &mockAlerter{}

	var err error
	s.notification, err = presenter.NewNotificationPresenter(s.notifQuerier, &stubMessageUpdater{})
	s.Require().NoError(err)

	s.activity, err = presenter.NewActivityPresenter(s.actSource, 500)
	s.Require().NoError(err)

	s.feedback, err = presenter.NewFeedbackPresenter(&stubBufferReviewer{})
	s.Require().NoError(err)
}

// --- Constructor Tests ---

func (s *AppPresenterSuite) TestConstructorRequiresNotificationPresenter() {
	_, err := presenter.NewAppPresenter(nil, s.activity, s.feedback, s.alerter)
	s.Error(err)
	s.Contains(err.Error(), "notification")
}

func (s *AppPresenterSuite) TestConstructorRequiresActivityPresenter() {
	_, err := presenter.NewAppPresenter(s.notification, nil, s.feedback, s.alerter)
	s.Error(err)
	s.Contains(err.Error(), "activity")
}

func (s *AppPresenterSuite) TestConstructorRequiresFeedbackPresenter() {
	_, err := presenter.NewAppPresenter(s.notification, s.activity, nil, s.alerter)
	s.Error(err)
	s.Contains(err.Error(), "feedback")
}

func (s *AppPresenterSuite) TestConstructorAllowsNilAlerter() {
	p, err := presenter.NewAppPresenter(s.notification, s.activity, s.feedback, nil)
	s.NoError(err)
	s.NotNil(p)
}

// --- Start Tests ---

func (s *AppPresenterSuite) TestStartPlaysStartupAlert() {
	p, err := presenter.NewAppPresenter(s.notification, s.activity, s.feedback, s.alerter)
	s.Require().NoError(err)

	ctx := context.Background()
	err = p.Start(ctx)
	s.Require().NoError(err)
	defer p.Shutdown(ctx) //nolint:errcheck

	s.True(s.alerter.startupCalled)
}

func (s *AppPresenterSuite) TestStartWithNilAlerterDoesNotPanic() {
	p, err := presenter.NewAppPresenter(s.notification, s.activity, s.feedback, nil)
	s.Require().NoError(err)

	ctx := context.Background()
	s.NotPanics(func() {
		err = p.Start(ctx)
	})
	s.NoError(err)
	defer p.Shutdown(ctx) //nolint:errcheck
}

func (s *AppPresenterSuite) TestStartStartsActivityPresenter() {
	p, err := presenter.NewAppPresenter(s.notification, s.activity, s.feedback, s.alerter)
	s.Require().NoError(err)

	ctx := context.Background()
	err = p.Start(ctx)
	s.Require().NoError(err)
	defer p.Shutdown(ctx) //nolint:errcheck

	// Send an event to the activity source; if Start started the activity
	// presenter, it should be consumed.
	s.actSource.ch <- presenter.ActivityEvent{Source: "test", Message: "hello"}

	s.Eventually(func() bool {
		return len(s.activity.Entries()) == 1
	}, time.Second, 10*time.Millisecond)
}

func (s *AppPresenterSuite) TestStartTriggersNotificationRefresh() {
	s.notifQuerier.messages = []*repository.Message{
		{
			Source:     "slack",
			Sender:     "alice",
			Channel:    "general",
			RawContent: "test message",
			Status:     "Notified",
		},
	}

	p, err := presenter.NewAppPresenter(s.notification, s.activity, s.feedback, s.alerter)
	s.Require().NoError(err)

	ctx := context.Background()
	err = p.Start(ctx)
	s.Require().NoError(err)
	defer p.Shutdown(ctx) //nolint:errcheck

	// After Start, notification presenter should have been refreshed
	// and contain the message from the querier.
	rows := s.notification.Messages()
	s.Len(rows, 1)
}

// --- Shutdown Tests ---

func (s *AppPresenterSuite) TestShutdownPlaysShutdownAlert() {
	p, err := presenter.NewAppPresenter(s.notification, s.activity, s.feedback, s.alerter)
	s.Require().NoError(err)

	ctx := context.Background()
	err = p.Start(ctx)
	s.Require().NoError(err)

	err = p.Shutdown(ctx)
	s.Require().NoError(err)

	s.True(s.alerter.shutdownCalled)
}

func (s *AppPresenterSuite) TestShutdownWithNilAlerterDoesNotPanic() {
	p, err := presenter.NewAppPresenter(s.notification, s.activity, s.feedback, nil)
	s.Require().NoError(err)

	ctx := context.Background()
	_ = p.Start(ctx)

	s.NotPanics(func() {
		err = p.Shutdown(ctx)
	})
	s.NoError(err)
}

func (s *AppPresenterSuite) TestShutdownStopsActivityPresenter() {
	p, err := presenter.NewAppPresenter(s.notification, s.activity, s.feedback, s.alerter)
	s.Require().NoError(err)

	ctx := context.Background()
	err = p.Start(ctx)
	s.Require().NoError(err)

	// Verify activity is running by sending an event.
	s.actSource.ch <- presenter.ActivityEvent{Source: "test", Message: "before"}

	s.Eventually(func() bool {
		return len(s.activity.Entries()) == 1
	}, time.Second, 10*time.Millisecond)

	// Shutdown should stop the activity presenter.
	err = p.Shutdown(ctx)
	s.Require().NoError(err)

	// Send another event after shutdown — should not be consumed.
	s.actSource.ch <- presenter.ActivityEvent{Source: "test", Message: "after"}

	time.Sleep(50 * time.Millisecond)

	s.Equal(1, len(s.activity.Entries()))
}
