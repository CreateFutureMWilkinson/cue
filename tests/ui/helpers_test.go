//go:build ui_acceptance

package ui_acceptance_test

import (
	"context"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/google/uuid"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/service/planner"
	"github.com/CreateFutureMWilkinson/cue/internal/ui"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

// --- Mock implementations ---

// mockQuerier returns pre-configured messages for notification tests.
type mockQuerier struct {
	messages []*repository.Message
}

func (m *mockQuerier) QueryByStatus(_ context.Context, _ string) ([]*repository.Message, error) {
	return m.messages, nil
}

// mockUpdater records Update calls.
type mockUpdater struct {
	updateCalled bool
	lastMessage  *repository.Message
}

func (m *mockUpdater) Update(_ context.Context, msg *repository.Message) error {
	m.updateCalled = true
	m.lastMessage = msg
	return nil
}

// mockBufferReviewer provides canned feedback review data.
type mockBufferReviewer struct {
	messages []*repository.Message
}

func (m *mockBufferReviewer) GetBufferedMessages(_ context.Context) ([]*repository.Message, error) {
	return m.messages, nil
}

func (m *mockBufferReviewer) CountBuffered(_ context.Context) (int, error) {
	return len(m.messages), nil
}

func (m *mockBufferReviewer) SaveRating(_ context.Context, _ uuid.UUID, _ int, _ *string) error {
	return nil
}

func (m *mockBufferReviewer) DeleteMessage(_ context.Context, _ uuid.UUID) error {
	return nil
}

// mockVolumeController records volume changes.
type mockVolumeController struct {
	volume int
}

func (m *mockVolumeController) SetVolume(v int) {
	m.volume = v
}

// mockServiceConfigRepo satisfies repository.ServiceConfigRepository.
// It can be pre-loaded with accounts and tracks upserted accounts.
type mockServiceConfigRepo struct {
	slackAccounts    []*repository.SlackAccount
	emailAccounts    []*repository.EmailAccount
	calendarAccounts []*repository.CalendarAccount
}

func (m *mockServiceConfigRepo) ListSlackAccounts(_ context.Context) ([]*repository.SlackAccount, error) {
	return m.slackAccounts, nil
}

func (m *mockServiceConfigRepo) GetSlackAccount(_ context.Context, _ uuid.UUID) (*repository.SlackAccount, error) {
	return nil, nil
}

func (m *mockServiceConfigRepo) UpsertSlackAccount(_ context.Context, acct *repository.SlackAccount) error {
	m.slackAccounts = append(m.slackAccounts, acct)
	return nil
}

func (m *mockServiceConfigRepo) DeleteSlackAccount(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockServiceConfigRepo) ListEmailAccounts(_ context.Context) ([]*repository.EmailAccount, error) {
	return m.emailAccounts, nil
}

func (m *mockServiceConfigRepo) GetEmailAccount(_ context.Context, _ uuid.UUID) (*repository.EmailAccount, error) {
	return nil, nil
}

func (m *mockServiceConfigRepo) UpsertEmailAccount(_ context.Context, acct *repository.EmailAccount) error {
	m.emailAccounts = append(m.emailAccounts, acct)
	return nil
}

func (m *mockServiceConfigRepo) DeleteEmailAccount(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockServiceConfigRepo) ListCalendarAccounts(_ context.Context) ([]*repository.CalendarAccount, error) {
	return m.calendarAccounts, nil
}

func (m *mockServiceConfigRepo) GetCalendarAccount(_ context.Context, _ uuid.UUID) (*repository.CalendarAccount, error) {
	return nil, nil
}

func (m *mockServiceConfigRepo) UpsertCalendarAccount(_ context.Context, acct *repository.CalendarAccount) error {
	m.calendarAccounts = append(m.calendarAccounts, acct)
	return nil
}

func (m *mockServiceConfigRepo) DeleteCalendarAccount(_ context.Context, _ uuid.UUID) error {
	return nil
}

// mockActivitySource satisfies presenter.ActivitySource with a no-op channel.
type mockActivitySource struct {
	ch chan presenter.ActivityEvent
}

func newMockActivitySource() *mockActivitySource {
	return &mockActivitySource{ch: make(chan presenter.ActivityEvent)}
}

func (m *mockActivitySource) Events() <-chan presenter.ActivityEvent {
	return m.ch
}

// mockWatcherRemover satisfies presenter.WatcherRemover.
type mockWatcherRemover struct{}

func (m *mockWatcherRemover) RemoveWatcher(_ string) {}

// stubPlannerTimerVM satisfies both PlannerViewModel and TimerViewModel.
type stubPlannerTimerVM struct {
	hasActivePlan  bool
	activeSchedule *presenter.ActiveScheduleState
	tasks          []presenter.TodoRow
	step           presenter.WizardStep
}

func (s *stubPlannerTimerVM) CurrentStep() presenter.WizardStep      { return s.step }
func (s *stubPlannerTimerVM) HasActivePlan() bool                    { return s.hasActivePlan }
func (s *stubPlannerTimerVM) AvailableTasks() []presenter.TodoRow    { return s.tasks }
func (s *stubPlannerTimerVM) Estimates() []presenter.TaskEstimateRow { return nil }
func (s *stubPlannerTimerVM) EstimateSummary() presenter.EstimateSummary {
	return presenter.EstimateSummary{}
}
func (s *stubPlannerTimerVM) FocusSchedule() *presenter.SchedulePreview    { return nil }
func (s *stubPlannerTimerVM) RecoverySchedule() *presenter.SchedulePreview { return nil }
func (s *stubPlannerTimerVM) ActiveSchedule() *presenter.ActiveScheduleState {
	return s.activeSchedule
}
func (s *stubPlannerTimerVM) IsRunning() bool              { return false }
func (s *stubPlannerTimerVM) ActiveSegment() int           { return 0 }
func (s *stubPlannerTimerVM) ElapsedFraction() float64     { return 0 }
func (s *stubPlannerTimerVM) IsFlashVisible() bool         { return false }
func (s *stubPlannerTimerVM) CurrentTaskName() string      { return "" }
func (s *stubPlannerTimerVM) BlockType() planner.BlockType { return planner.BlockFocus }

// TodoListViewModel implementation for stubPlannerTimerVM.
func (s *stubPlannerTimerVM) AllTodos() []ui.TodoListRow  { return nil }
func (s *stubPlannerTimerVM) ToggleComplete(_ uuid.UUID)  {}
func (s *stubPlannerTimerVM) AddTask(_ string, _ int)     {}
func (s *stubPlannerTimerVM) UpdateTask(_ ui.TodoListRow) {}

// stubWizardVM satisfies WizardViewModel.
type stubWizardVM struct {
	step          presenter.WizardStep
	tasks         []presenter.TodoRow
	estimates     []presenter.TaskEstimateRow
	summary       presenter.EstimateSummary
	selectedCount int
}

func (s *stubWizardVM) CurrentStep() presenter.WizardStep      { return s.step }
func (s *stubWizardVM) AvailableTasks() []presenter.TodoRow    { return s.tasks }
func (s *stubWizardVM) Estimates() []presenter.TaskEstimateRow { return s.estimates }
func (s *stubWizardVM) EstimateSummary() presenter.EstimateSummary {
	return s.summary
}
func (s *stubWizardVM) FocusSchedule() *presenter.SchedulePreview        { return nil }
func (s *stubWizardVM) RecoverySchedule() *presenter.SchedulePreview     { return nil }
func (s *stubWizardVM) SelectTask(_ uuid.UUID, _ bool)                   {}
func (s *stubWizardVM) AddTask(_ context.Context, _ string, _ int) error { return nil }
func (s *stubWizardVM) NextStep(_ context.Context) error                 { return nil }
func (s *stubWizardVM) PreviousStep()                                    {}
func (s *stubWizardVM) OverrideEstimate(_ uuid.UUID, _ int)              {}
func (s *stubWizardVM) ReorderTask(_, _ int)                             {}
func (s *stubWizardVM) SelectSchedule(_ context.Context, _ string) error { return nil }
func (s *stubWizardVM) SelectedCount() int                               { return s.selectedCount }

// mockRoutingRuleRepo satisfies repository.RoutingRuleRepository with in-memory storage.
type mockRoutingRuleRepo struct {
	rules []*repository.RoutingRule
}

func (m *mockRoutingRuleRepo) ListRules(_ context.Context) ([]*repository.RoutingRule, error) {
	return m.rules, nil
}

func (m *mockRoutingRuleRepo) ListRulesBySourceType(_ context.Context, sourceType string) ([]*repository.RoutingRule, error) {
	var result []*repository.RoutingRule
	for _, r := range m.rules {
		if r.SourceType == sourceType {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *mockRoutingRuleRepo) ListRulesBySourceAccount(_ context.Context, accountID uuid.UUID) ([]*repository.RoutingRule, error) {
	var result []*repository.RoutingRule
	for _, r := range m.rules {
		if r.SourceAccount != nil && *r.SourceAccount == accountID {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *mockRoutingRuleRepo) GetRule(_ context.Context, id uuid.UUID) (*repository.RoutingRule, error) {
	for _, r := range m.rules {
		if r.ID == id {
			return r, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (m *mockRoutingRuleRepo) UpsertRule(_ context.Context, rule *repository.RoutingRule) error {
	for i, r := range m.rules {
		if r.ID == rule.ID {
			m.rules[i] = rule
			return nil
		}
	}
	m.rules = append(m.rules, rule)
	return nil
}

func (m *mockRoutingRuleRepo) DeleteRule(_ context.Context, id uuid.UUID) error {
	for i, r := range m.rules {
		if r.ID == id {
			m.rules = append(m.rules[:i], m.rules[i+1:]...)
			return nil
		}
	}
	return nil
}

// mockQueueRepo satisfies repository.QueueRepository.
type mockQueueRepo struct {
	pending int
}

func (m *mockQueueRepo) Enqueue(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockQueueRepo) DequeueOldest(_ context.Context) (*repository.QueueEntry, error) {
	return nil, nil
}
func (m *mockQueueRepo) MarkDone(_ context.Context, _ uuid.UUID) error       { return nil }
func (m *mockQueueRepo) MarkFailed(_ context.Context, _ uuid.UUID) error     { return nil }
func (m *mockQueueRepo) PendingCount(_ context.Context) (int, error)         { return m.pending, nil }
func (m *mockQueueRepo) PurgeCompleted(_ context.Context) error              { return nil }
func (m *mockQueueRepo) PurgeOlderThan(_ context.Context, _ time.Time) error { return nil }
func (m *mockQueueRepo) PurgeAll(_ context.Context) error                    { return nil }
func (m *mockQueueRepo) ResetProcessing(_ context.Context) (int64, error)    { return 0, nil }

// --- Factory functions ---

// defaultGUIConfig returns a standard GUIConfig for acceptance tests.
func defaultGUIConfig() config.GUIConfig {
	return config.GUIConfig{
		WindowWidth:  1200,
		WindowHeight: 800,
	}
}

// defaultOllamaConfig returns a standard OllamaConfig for acceptance tests.
func defaultOllamaConfig() config.OllamaConfig {
	return config.OllamaConfig{
		Host:           "localhost",
		Port:           11434,
		InferenceModel: "neural-chat",
		EmbeddingModel: "nomic-embed-text",
		TimeoutSeconds: 10,
	}
}

// newMinimalMainWindow creates a MainWindow with nil presenters for layout tests.
func newMinimalMainWindow(fyneApp fyne.App, router *ui.CenterViewRouter) *ui.MainWindow {
	return ui.NewMainWindow(
		fyneApp,
		defaultGUIConfig(),
		(*presenter.NotificationPresenter)(nil),
		(*presenter.ActivityPresenter)(nil),
		(*presenter.FeedbackPresenter)(nil),
		(*presenter.AppPresenter)(nil),
		(*presenter.SettingsPresenter)(nil),
		(*presenter.ServiceSettingsPresenter)(nil),
		nil, // rp
		defaultOllamaConfig(),
		nil, // characterWidget
		router,
		nil, // plannerVM
		nil, // timerVM
		nil, // wizardVM
		nil, // rightPanelOverride
	)
}

// newMainWindowWithNotifications creates a MainWindow with a notification presenter
// backed by sample messages.
func newMainWindowWithNotifications(fyneApp fyne.App, router *ui.CenterViewRouter, messages []*repository.Message) (*ui.MainWindow, *presenter.NotificationPresenter) {
	querier := &mockQuerier{messages: messages}
	updater := &mockUpdater{}
	np, _ := presenter.NewNotificationPresenter(querier, updater)
	_ = np.Refresh(context.Background())

	mw := ui.NewMainWindow(
		fyneApp,
		defaultGUIConfig(),
		np,
		(*presenter.ActivityPresenter)(nil),
		(*presenter.FeedbackPresenter)(nil),
		(*presenter.AppPresenter)(nil),
		(*presenter.SettingsPresenter)(nil),
		(*presenter.ServiceSettingsPresenter)(nil),
		nil, // rp
		defaultOllamaConfig(),
		nil, // characterWidget
		router,
		nil, // plannerVM
		nil, // timerVM
		nil, // wizardVM
		nil, // rightPanelOverride
	)
	return mw, np
}

// newMainWindowWithFeedback creates a MainWindow with notification + feedback presenters.
func newMainWindowWithFeedback(fyneApp fyne.App, router *ui.CenterViewRouter, notifications []*repository.Message, buffered []*repository.Message) (*ui.MainWindow, *presenter.NotificationPresenter, *presenter.FeedbackPresenter) {
	querier := &mockQuerier{messages: notifications}
	updater := &mockUpdater{}
	np, _ := presenter.NewNotificationPresenter(querier, updater)
	_ = np.Refresh(context.Background())

	reviewer := &mockBufferReviewer{messages: buffered}
	fp, _ := presenter.NewFeedbackPresenter(reviewer)

	mw := ui.NewMainWindow(
		fyneApp,
		defaultGUIConfig(),
		np,
		(*presenter.ActivityPresenter)(nil),
		fp,
		(*presenter.AppPresenter)(nil),
		(*presenter.SettingsPresenter)(nil),
		(*presenter.ServiceSettingsPresenter)(nil),
		nil, // rp
		defaultOllamaConfig(),
		nil, // characterWidget
		router,
		nil, // plannerVM
		nil, // timerVM
		nil, // wizardVM
		nil, // rightPanelOverride
	)
	return mw, np, fp
}

// newMainWindowWithPlanner creates a MainWindow with planner and timer VMs.
func newMainWindowWithPlanner(fyneApp fyne.App, router *ui.CenterViewRouter, plannerVM *stubPlannerTimerVM) *ui.MainWindow {
	return ui.NewMainWindow(
		fyneApp,
		defaultGUIConfig(),
		(*presenter.NotificationPresenter)(nil),
		(*presenter.ActivityPresenter)(nil),
		(*presenter.FeedbackPresenter)(nil),
		(*presenter.AppPresenter)(nil),
		(*presenter.SettingsPresenter)(nil),
		(*presenter.ServiceSettingsPresenter)(nil),
		nil, // rp
		defaultOllamaConfig(),
		nil, // characterWidget
		router,
		plannerVM, // plannerVM
		plannerVM, // timerVM
		nil,       // wizardVM
		nil,       // rightPanelOverride
	)
}

// newMainWindowWithWizard creates a MainWindow with a wizard VM.
func newMainWindowWithWizard(fyneApp fyne.App, router *ui.CenterViewRouter, wizardVM *stubWizardVM) *ui.MainWindow {
	return ui.NewMainWindow(
		fyneApp,
		defaultGUIConfig(),
		(*presenter.NotificationPresenter)(nil),
		(*presenter.ActivityPresenter)(nil),
		(*presenter.FeedbackPresenter)(nil),
		(*presenter.AppPresenter)(nil),
		(*presenter.SettingsPresenter)(nil),
		(*presenter.ServiceSettingsPresenter)(nil),
		nil, // rp
		defaultOllamaConfig(),
		nil, // characterWidget
		router,
		nil,      // plannerVM
		nil,      // timerVM
		wizardVM, // wizardVM
		nil,      // rightPanelOverride
	)
}

// newSettingsView creates a SettingsView with real presenters backed by mocks.
func newSettingsView() *ui.SettingsView {
	vc := &mockVolumeController{}
	sp, _ := presenter.NewSettingsPresenter(vc, 50, &mockVolumeController{}, 50)
	ssp := presenter.NewServiceSettingsPresenter(&mockServiceConfigRepo{}, &mockWatcherRemover{}, func(_ string, _ uuid.UUID) error { return nil })
	return ui.NewSettingsView(sp, ssp, nil, defaultOllamaConfig(), func() {})
}

// newSettingsViewWithRepo creates a SettingsView backed by a specific mock repo.
func newSettingsViewWithRepo(repo *mockServiceConfigRepo, opts ...presenter.ServiceSettingsOption) *ui.SettingsView {
	vc := &mockVolumeController{}
	sp, _ := presenter.NewSettingsPresenter(vc, 50, &mockVolumeController{}, 50)
	ssp := presenter.NewServiceSettingsPresenter(repo, &mockWatcherRemover{}, func(_ string, _ uuid.UUID) error { return nil }, opts...)
	return ui.NewSettingsView(sp, ssp, nil, defaultOllamaConfig(), func() {})
}

// newSettingsViewWithRules creates a SettingsView with a RulesPresenter backed by mocks.
func newSettingsViewWithRules(ruleRepo *mockRoutingRuleRepo, queueRepo *mockQueueRepo, warnAt int) *ui.SettingsView {
	vc := &mockVolumeController{}
	sp, _ := presenter.NewSettingsPresenter(vc, 50, &mockVolumeController{}, 50)
	ssp := presenter.NewServiceSettingsPresenter(&mockServiceConfigRepo{}, &mockWatcherRemover{}, func(_ string, _ uuid.UUID) error { return nil })
	rp := presenter.NewRulesPresenter(ruleRepo, queueRepo, warnAt)
	return ui.NewSettingsView(sp, ssp, rp, defaultOllamaConfig(), func() {})
}

// --- Mock validators ---

// mockSlackValidator is a mock SlackValidator that returns a configurable error.
type mockSlackValidator struct {
	err error
}

func (m *mockSlackValidator) ValidateSlack(_ context.Context, _ string) error {
	return m.err
}

// mockEmailValidator is a mock EmailValidator that returns a configurable error.
type mockEmailValidator struct {
	err error
}

func (m *mockEmailValidator) ValidateEmail(_ context.Context, _ string, _ int, _, _, _ string) error {
	return m.err
}

// mockCalendarValidator is a mock CalendarValidator that returns a configurable error.
type mockCalendarValidator struct {
	err error
}

func (m *mockCalendarValidator) ValidateCalendar(_ context.Context, _ string) error {
	return m.err
}

// --- Sample data ---

// sampleNotifiedMessages returns a set of messages at various IS levels.
func sampleNotifiedMessages() []*repository.Message {
	return []*repository.Message{
		{
			ID:              uuid.New(),
			Source:          "slack",
			Sender:          "alice",
			Channel:         "general",
			RawContent:      "Server is on fire! Everything is down!",
			ImportanceScore: 9.5,
			ConfidenceScore: 0.95,
			Reasoning:       "Critical server outage",
			Status:          "Notified",
			CreatedAt:       time.Now().Add(-2 * time.Minute),
		},
		{
			ID:              uuid.New(),
			Source:          "email",
			Sender:          "bob@example.com",
			Channel:         "inbox",
			RawContent:      "Quarterly report deadline is tomorrow",
			ImportanceScore: 8.0,
			ConfidenceScore: 0.85,
			Reasoning:       "Upcoming deadline",
			Status:          "Notified",
			CreatedAt:       time.Now().Add(-5 * time.Minute),
		},
		{
			ID:              uuid.New(),
			Source:          "slack",
			Sender:          "carol",
			Channel:         "team-updates",
			RawContent:      "Meeting rescheduled to next week",
			ImportanceScore: 7.0,
			ConfidenceScore: 0.80,
			Reasoning:       "Schedule change",
			Status:          "Notified",
			CreatedAt:       time.Now().Add(-10 * time.Minute),
		},
	}
}

// sampleBufferedMessages returns messages in Buffered status for feedback review.
func sampleBufferedMessages() []*repository.Message {
	return []*repository.Message{
		{
			ID:              uuid.New(),
			Source:          "slack",
			Sender:          "dave",
			Channel:         "random",
			RawContent:      "Anyone for lunch?",
			ImportanceScore: 7.0,
			ConfidenceScore: 0.5,
			Reasoning:       "Social, low confidence",
			Status:          "Buffered",
			CreatedAt:       time.Now().Add(-15 * time.Minute),
		},
		{
			ID:              uuid.New(),
			Source:          "email",
			Sender:          "eve@example.com",
			Channel:         "inbox",
			RawContent:      "Newsletter: Top 10 tips for productivity",
			ImportanceScore: 7.5,
			ConfidenceScore: 0.6,
			Reasoning:       "Possibly relevant",
			Status:          "Buffered",
			CreatedAt:       time.Now().Add(-20 * time.Minute),
		},
	}
}

// newTestApp creates a headless Fyne test app.
func newTestApp() fyne.App {
	return test.NewApp()
}
