// Package main implements cue-fake, a UI-testing harness that mimics the
// cue-server HTTP/WebSocket API with in-memory backends. This file holds the
// in-memory implementations of every server.Deps interface.
package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/server/handler"
	"github.com/CreateFutureMWilkinson/cue/internal/service/calendar"
	"github.com/CreateFutureMWilkinson/cue/internal/service/planner"
	"github.com/CreateFutureMWilkinson/cue/internal/service/servicemanager"
	"github.com/google/uuid"
)

// store is the single in-memory backing struct used by all stub services.
// One mutex protects everything; the harness is single-process and not
// performance-sensitive, so coarse-grained locking keeps the code small.
type store struct {
	mu sync.Mutex

	messages   []*repository.Message
	tasks      []*repository.Task
	categories []*repository.Category
	rules      []*repository.RoutingRule
	schedules  map[string]*repository.Schedule // keyed by YYYY-MM-DD

	slack    []*repository.SlackAccount
	emails   []*repository.EmailAccount
	calendar []*repository.CalendarAccount

	tokens []*repository.AuthToken
}

func newStore() *store {
	return &store{schedules: make(map[string]*repository.Schedule)}
}

// reset clears every collection (does NOT reseed; caller does that).
func (s *store) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = nil
	s.tasks = nil
	s.categories = nil
	s.rules = nil
	s.schedules = make(map[string]*repository.Schedule)
	s.slack = nil
	s.emails = nil
	s.calendar = nil
	s.tokens = nil
}

// webURLFor returns the configured account WebURL for an injected message.
// Matches Slack accounts by WorkspaceID or FriendlyName, Email accounts by
// Username or FriendlyName, and returns "" when no account is found — the
// caller may also pass an explicit override that takes precedence.
func (s *store) webURLFor(source, account string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch source {
	case "slack":
		for _, acct := range s.slack {
			if acct.WorkspaceID == account || acct.FriendlyName == account {
				return acct.WebURL
			}
		}
	case "email":
		for _, acct := range s.emails {
			if acct.Username == account || acct.FriendlyName == account {
				return acct.WebURL
			}
		}
	}
	return ""
}

// snapshot returns a shallow JSON-friendly representation of current state.
func (s *store) snapshot() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return map[string]any{
		"messages":   len(s.messages),
		"tasks":      len(s.tasks),
		"categories": len(s.categories),
		"rules":      len(s.rules),
		"schedules":  len(s.schedules),
		"slack":      s.slack,
		"emails":     s.emails,
		"calendar":   s.calendar,
		"tokens":     len(s.tokens),
	}
}

// ---- MessageQuerier (handler.MessageQuerier) + BufferRater ----

type messageSvc struct{ s *store }

func (m *messageSvc) QueryFiltered(_ context.Context, f repository.MessageFilter) ([]*repository.Message, int, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	var matched []*repository.Message
	for _, msg := range m.s.messages {
		if f.Status != "" && msg.Status != f.Status {
			continue
		}
		if f.Source != "" && msg.Source != f.Source {
			continue
		}
		if f.Channel != "" && msg.Channel != f.Channel {
			continue
		}
		if f.Since != nil && msg.CreatedAt.Before(*f.Since) {
			continue
		}
		matched = append(matched, msg)
	}
	total := len(matched)
	off := min(f.Offset, total)
	end := off + f.Limit
	if f.Limit <= 0 || end > total {
		end = total
	}
	return matched[off:end], total, nil
}

func (m *messageSvc) QueryByID(_ context.Context, id uuid.UUID) (*repository.Message, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	for _, msg := range m.s.messages {
		if msg.ID == id {
			return msg, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (m *messageSvc) Update(_ context.Context, msg *repository.Message) error {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	for i, existing := range m.s.messages {
		if existing.ID == msg.ID {
			m.s.messages[i] = msg
			return nil
		}
	}
	return repository.ErrNotFound
}

// BufferRater
func (m *messageSvc) SaveRating(_ context.Context, id uuid.UUID, rating int, feedback *string) error {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	for _, msg := range m.s.messages {
		if msg.ID == id {
			r := rating
			msg.UserRating = &r
			msg.UserFeedback = feedback
			msg.Status = "Resolved"
			now := time.Now().UTC()
			msg.UpdatedAt = now
			msg.ResolvedAt = &now
			return nil
		}
	}
	return repository.ErrNotFound
}

func (m *messageSvc) DeleteMessage(_ context.Context, id uuid.UUID) error {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	for _, msg := range m.s.messages {
		if msg.ID == id {
			msg.Status = "Resolved"
			now := time.Now().UTC()
			msg.UpdatedAt = now
			msg.ResolvedAt = &now
			return nil
		}
	}
	return repository.ErrNotFound
}

// ---- TaskServicer ----

type taskSvc struct{ s *store }

func (t *taskSvc) Create(_ context.Context, task *repository.Task) (*repository.Task, error) {
	t.s.mu.Lock()
	defer t.s.mu.Unlock()
	if task.ID == uuid.Nil {
		task.ID = uuid.New()
	}
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now().UTC()
	}
	t.s.tasks = append(t.s.tasks, task)
	return task, nil
}

func (t *taskSvc) Get(_ context.Context, id uuid.UUID) (*repository.Task, error) {
	t.s.mu.Lock()
	defer t.s.mu.Unlock()
	for _, x := range t.s.tasks {
		if x.ID == id {
			return x, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (t *taskSvc) List(_ context.Context, f repository.TaskFilter) ([]*repository.Task, int, error) {
	t.s.mu.Lock()
	defer t.s.mu.Unlock()
	var matched []*repository.Task
	for _, x := range t.s.tasks {
		switch f.Status {
		case "completed", "complete":
			if x.CompletedAt == nil {
				continue
			}
		case "incomplete", "open", "":
			if x.CompletedAt != nil {
				continue
			}
		}
		if f.CategoryKey != "" {
			if x.CategoryKey == nil || *x.CategoryKey != f.CategoryKey {
				continue
			}
		}
		if f.Search != "" {
			needle := strings.ToLower(f.Search)
			if !strings.Contains(strings.ToLower(x.Title), needle) &&
				!strings.Contains(strings.ToLower(x.Description), needle) {
				continue
			}
		}
		matched = append(matched, x)
	}
	total := len(matched)
	off := min(f.Offset, total)
	end := off + f.Limit
	if f.Limit <= 0 || end > total {
		end = total
	}
	return matched[off:end], total, nil
}

func (t *taskSvc) Update(_ context.Context, task *repository.Task) (*repository.Task, error) {
	t.s.mu.Lock()
	defer t.s.mu.Unlock()
	for i, x := range t.s.tasks {
		if x.ID == task.ID {
			t.s.tasks[i] = task
			return task, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (t *taskSvc) Delete(_ context.Context, id uuid.UUID) error {
	t.s.mu.Lock()
	defer t.s.mu.Unlock()
	for i, x := range t.s.tasks {
		if x.ID == id {
			t.s.tasks = append(t.s.tasks[:i], t.s.tasks[i+1:]...)
			return nil
		}
	}
	return repository.ErrNotFound
}

// effectiveEstimate: prefer user-supplied; fall back to LLM.
func effectiveEstimate(t *repository.Task) *int {
	if t.EstimateMinutes != nil {
		return t.EstimateMinutes
	}
	return t.LLMEstimateMinutes
}

// ---- CategoryServicer ----

type categorySvc struct{ s *store }

func (c *categorySvc) CreateCategory(_ context.Context, raw string, colour *string) (*repository.Category, error) {
	key, err := repository.NormalizeCategoryKey(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", repository.ErrValidation, err)
	}
	c.s.mu.Lock()
	defer c.s.mu.Unlock()
	for _, x := range c.s.categories {
		if x.NameKey == key {
			return nil, repository.ErrDuplicate
		}
	}
	cat := &repository.Category{NameKey: key, Colour: colour, CreatedAt: time.Now().UTC()}
	c.s.categories = append(c.s.categories, cat)
	return cat, nil
}

func (c *categorySvc) RenameCategory(_ context.Context, oldKey, newRaw string) (*repository.Category, error) {
	newKey, err := repository.NormalizeCategoryKey(newRaw)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", repository.ErrValidation, err)
	}
	c.s.mu.Lock()
	defer c.s.mu.Unlock()
	for _, x := range c.s.categories {
		if x.NameKey == newKey && x.NameKey != oldKey {
			return nil, repository.ErrDuplicate
		}
	}
	for _, x := range c.s.categories {
		if x.NameKey == oldKey {
			x.NameKey = newKey
			// Update task references
			for _, t := range c.s.tasks {
				if t.CategoryKey != nil && *t.CategoryKey == oldKey {
					t.CategoryKey = &newKey
				}
			}
			return x, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (c *categorySvc) SetCategoryColour(_ context.Context, key string, colour *string) error {
	c.s.mu.Lock()
	defer c.s.mu.Unlock()
	for _, x := range c.s.categories {
		if x.NameKey == key {
			x.Colour = colour
			return nil
		}
	}
	return repository.ErrNotFound
}

func (c *categorySvc) DeleteCategory(_ context.Context, key string) error {
	c.s.mu.Lock()
	defer c.s.mu.Unlock()
	for i, x := range c.s.categories {
		if x.NameKey == key {
			c.s.categories = append(c.s.categories[:i], c.s.categories[i+1:]...)
			return nil
		}
	}
	return repository.ErrNotFound
}

func (c *categorySvc) GetCategory(_ context.Context, raw string) (*repository.Category, error) {
	// Accept both raw and already-normalized input; try direct match first.
	c.s.mu.Lock()
	defer c.s.mu.Unlock()
	for _, x := range c.s.categories {
		if x.NameKey == raw {
			return x, nil
		}
	}
	if key, err := repository.NormalizeCategoryKey(raw); err == nil {
		for _, x := range c.s.categories {
			if x.NameKey == key {
				return x, nil
			}
		}
	}
	return nil, repository.ErrNotFound
}

func (c *categorySvc) ListCategories(_ context.Context, _ bool) ([]*repository.CategoryWithCount, error) {
	c.s.mu.Lock()
	defer c.s.mu.Unlock()
	out := make([]*repository.CategoryWithCount, 0, len(c.s.categories))
	for _, x := range c.s.categories {
		count := 0
		for _, t := range c.s.tasks {
			if t.CategoryKey != nil && *t.CategoryKey == x.NameKey {
				count++
			}
		}
		out = append(out, &repository.CategoryWithCount{Category: *x, TaskCount: count})
	}
	return out, nil
}

// ---- ScheduleStore ----

type scheduleSvc struct{ s *store }

func dateKey(t time.Time) string { return t.Format("2006-01-02") }

func (sc *scheduleSvc) LoadByDate(_ context.Context, date time.Time) (*repository.Schedule, error) {
	sc.s.mu.Lock()
	defer sc.s.mu.Unlock()
	if v, ok := sc.s.schedules[dateKey(date)]; ok {
		return v, nil
	}
	return nil, repository.ErrNotFound
}

func (sc *scheduleSvc) Save(_ context.Context, schedule *repository.Schedule) error {
	sc.s.mu.Lock()
	defer sc.s.mu.Unlock()
	sc.s.schedules[dateKey(schedule.Date)] = schedule
	return nil
}

func (sc *scheduleSvc) Delete(_ context.Context, date time.Time) error {
	sc.s.mu.Lock()
	defer sc.s.mu.Unlock()
	delete(sc.s.schedules, dateKey(date))
	return nil
}

// ---- ScheduleGenerator + CalendarFetcher (no-op stubs).
// These produce empty options. The /planner/generate endpoint will only
// be registered if both are non-nil; we return them so the route exists
// for clients but always returns empty schedules. Documented no-ops. ----

type noopScheduleGen struct{}

// GenerateSchedules returns two empty DaySchedules. Intentional no-op:
// the harness has no real planner; the GUI may call this endpoint and we
// want a valid empty response rather than 404.
func (noopScheduleGen) GenerateSchedules(_ context.Context, _ []planner.TaskEstimate, _ []calendar.CalendarEvent, target time.Time) (*planner.DaySchedule, *planner.DaySchedule, error) {
	return &planner.DaySchedule{Date: target, Strategy: "focus-maximized"},
		&planner.DaySchedule{Date: target, Strategy: "recovery-balanced"},
		nil
}

func (noopScheduleGen) TargetDate(now time.Time) time.Time {
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

type noopCalendar struct{}

// FetchEvents returns no events. Intentional no-op: harness has no real
// calendar feed.
func (noopCalendar) FetchEvents(_ context.Context, _ time.Time) ([]calendar.CalendarEvent, error) {
	return nil, nil
}

// ---- RulesManager ----

type rulesSvc struct{ s *store }

func (r *rulesSvc) ListRules(_ context.Context) ([]*repository.RoutingRule, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	out := make([]*repository.RoutingRule, len(r.s.rules))
	copy(out, r.s.rules)
	return out, nil
}

func (r *rulesSvc) ListRulesBySourceType(_ context.Context, st string) ([]*repository.RoutingRule, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	var out []*repository.RoutingRule
	for _, x := range r.s.rules {
		if x.SourceType == st {
			out = append(out, x)
		}
	}
	return out, nil
}

func (r *rulesSvc) ListRulesBySourceAccount(_ context.Context, id uuid.UUID) ([]*repository.RoutingRule, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	var out []*repository.RoutingRule
	for _, x := range r.s.rules {
		if x.SourceAccount != nil && *x.SourceAccount == id {
			out = append(out, x)
		}
	}
	return out, nil
}

func (r *rulesSvc) GetRule(_ context.Context, id uuid.UUID) (*repository.RoutingRule, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	for _, x := range r.s.rules {
		if x.ID == id {
			return x, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (r *rulesSvc) SaveRule(_ context.Context, rule *repository.RoutingRule) error {
	if err := rule.Validate(); err != nil {
		return err
	}
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	for i, x := range r.s.rules {
		if x.ID == rule.ID {
			r.s.rules[i] = rule
			return nil
		}
	}
	r.s.rules = append(r.s.rules, rule)
	return nil
}

func (r *rulesSvc) DeleteRule(_ context.Context, id uuid.UUID) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	for i, x := range r.s.rules {
		if x.ID == id {
			r.s.rules = append(r.s.rules[:i], r.s.rules[i+1:]...)
			return nil
		}
	}
	return nil
}

func (r *rulesSvc) ReorderRule(_ context.Context, id uuid.UUID, p int) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	for _, x := range r.s.rules {
		if x.ID == id {
			x.Priority = p
			return nil
		}
	}
	return errors.New("rule not found")
}

func (r *rulesSvc) ToggleRule(_ context.Context, id uuid.UUID, enabled bool) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	for _, x := range r.s.rules {
		if x.ID == id {
			x.Enabled = enabled
			return nil
		}
	}
	return errors.New("rule not found")
}

// ---- ServiceManager ----

type serviceMgr struct{ s *store }

func (m *serviceMgr) ListSlackAccounts(_ context.Context) ([]*repository.SlackAccount, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	out := make([]*repository.SlackAccount, len(m.s.slack))
	copy(out, m.s.slack)
	return out, nil
}
func (m *serviceMgr) ListEmailAccounts(_ context.Context) ([]*repository.EmailAccount, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	out := make([]*repository.EmailAccount, len(m.s.emails))
	copy(out, m.s.emails)
	return out, nil
}
func (m *serviceMgr) ListCalendarAccounts(_ context.Context) ([]*repository.CalendarAccount, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	out := make([]*repository.CalendarAccount, len(m.s.calendar))
	copy(out, m.s.calendar)
	return out, nil
}

func (m *serviceMgr) GetSlackAccount(_ context.Context, id uuid.UUID) (*repository.SlackAccount, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	for _, x := range m.s.slack {
		if x.ID == id {
			return x, nil
		}
	}
	return nil, fmt.Errorf("slack account not found")
}
func (m *serviceMgr) GetEmailAccount(_ context.Context, id uuid.UUID) (*repository.EmailAccount, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	for _, x := range m.s.emails {
		if x.ID == id {
			return x, nil
		}
	}
	return nil, fmt.Errorf("email account not found")
}
func (m *serviceMgr) GetCalendarAccount(_ context.Context, id uuid.UUID) (*repository.CalendarAccount, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	for _, x := range m.s.calendar {
		if x.ID == id {
			return x, nil
		}
	}
	return nil, fmt.Errorf("calendar account not found")
}

func (m *serviceMgr) CreateSlackAccount(_ context.Context, a *repository.SlackAccount) (*repository.SlackAccount, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	now := time.Now().UTC()
	a.CreatedAt = now
	a.UpdatedAt = now
	m.s.slack = append(m.s.slack, a)
	return a, nil
}
func (m *serviceMgr) CreateEmailAccount(_ context.Context, a *repository.EmailAccount) (*repository.EmailAccount, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	now := time.Now().UTC()
	a.CreatedAt = now
	a.UpdatedAt = now
	m.s.emails = append(m.s.emails, a)
	return a, nil
}
func (m *serviceMgr) CreateCalendarAccount(_ context.Context, a *repository.CalendarAccount) (*repository.CalendarAccount, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	now := time.Now().UTC()
	a.CreatedAt = now
	a.UpdatedAt = now
	m.s.calendar = append(m.s.calendar, a)
	return a, nil
}

func (m *serviceMgr) UpdateSlackAccount(_ context.Context, id uuid.UUID, a *repository.SlackAccount) (*repository.SlackAccount, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	for i, x := range m.s.slack {
		if x.ID == id {
			a.ID = id
			a.CreatedAt = x.CreatedAt
			a.UpdatedAt = time.Now().UTC()
			m.s.slack[i] = a
			return a, nil
		}
	}
	return nil, fmt.Errorf("slack account not found")
}
func (m *serviceMgr) UpdateEmailAccount(_ context.Context, id uuid.UUID, a *repository.EmailAccount) (*repository.EmailAccount, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	for i, x := range m.s.emails {
		if x.ID == id {
			a.ID = id
			a.CreatedAt = x.CreatedAt
			a.UpdatedAt = time.Now().UTC()
			m.s.emails[i] = a
			return a, nil
		}
	}
	return nil, fmt.Errorf("email account not found")
}
func (m *serviceMgr) UpdateCalendarAccount(_ context.Context, id uuid.UUID, a *repository.CalendarAccount) (*repository.CalendarAccount, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	for i, x := range m.s.calendar {
		if x.ID == id {
			a.ID = id
			a.CreatedAt = x.CreatedAt
			a.UpdatedAt = time.Now().UTC()
			m.s.calendar[i] = a
			return a, nil
		}
	}
	return nil, fmt.Errorf("calendar account not found")
}

func (m *serviceMgr) DeleteSlackAccount(_ context.Context, id uuid.UUID) error {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	for i, x := range m.s.slack {
		if x.ID == id {
			m.s.slack = append(m.s.slack[:i], m.s.slack[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("slack account not found")
}
func (m *serviceMgr) DeleteEmailAccount(_ context.Context, id uuid.UUID) error {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	for i, x := range m.s.emails {
		if x.ID == id {
			m.s.emails = append(m.s.emails[:i], m.s.emails[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("email account not found")
}
func (m *serviceMgr) DeleteCalendarAccount(_ context.Context, id uuid.UUID) error {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	for i, x := range m.s.calendar {
		if x.ID == id {
			m.s.calendar = append(m.s.calendar[:i], m.s.calendar[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("calendar account not found")
}

func (m *serviceMgr) ToggleSlackAccount(_ context.Context, id uuid.UUID, enabled bool) error {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	for _, x := range m.s.slack {
		if x.ID == id {
			x.Enabled = enabled
			return nil
		}
	}
	return fmt.Errorf("slack account not found")
}
func (m *serviceMgr) ToggleEmailAccount(_ context.Context, id uuid.UUID, enabled bool) error {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	for _, x := range m.s.emails {
		if x.ID == id {
			x.Enabled = enabled
			return nil
		}
	}
	return fmt.Errorf("email account not found")
}
func (m *serviceMgr) ToggleCalendarAccount(_ context.Context, id uuid.UUID, enabled bool) error {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	for _, x := range m.s.calendar {
		if x.ID == id {
			x.Enabled = enabled
			return nil
		}
	}
	return fmt.Errorf("calendar account not found")
}

func (m *serviceMgr) Status(_ context.Context) ([]servicemanager.ServiceStatus, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	out := []servicemanager.ServiceStatus{}
	for _, x := range m.s.slack {
		name := x.FriendlyName
		if name == "" {
			name = x.WorkspaceID
		}
		out = append(out, servicemanager.ServiceStatus{
			ID: x.ID, Type: "slack", Name: name, Enabled: x.Enabled, WatcherRegistered: x.Enabled,
		})
	}
	for _, x := range m.s.emails {
		name := x.FriendlyName
		if name == "" {
			name = x.Username
		}
		out = append(out, servicemanager.ServiceStatus{
			ID: x.ID, Type: "email", Name: name, Enabled: x.Enabled, WatcherRegistered: x.Enabled,
		})
	}
	for _, x := range m.s.calendar {
		out = append(out, servicemanager.ServiceStatus{
			ID: x.ID, Type: "calendar", Name: x.Name, Enabled: x.Enabled, WatcherRegistered: false,
		})
	}
	return out, nil
}

// ---- AuthTokenManager + AuthTokenLookup ----

type tokenMgr struct{ s *store }

// LookupByHash satisfies server.AuthTokenLookup. The fake harness accepts
// every bearer token: it returns a synthetic AuthToken regardless of input.
func (t *tokenMgr) LookupByHash(_ context.Context, _ string) (*repository.AuthToken, error) {
	return &repository.AuthToken{
		ID: uuid.New(), Label: "fake", TokenHash: "fake",
		CreatedAt: time.Now(), LastSeen: time.Now(),
	}, nil
}

// UpdateLastSeen is a no-op: the fake harness does not track per-token usage.
func (t *tokenMgr) UpdateLastSeen(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}

// CountActive: the harness reports zero so the auth middleware treats us as a
// fresh deployment and auto-issues to the first client (defensive — auth is
// disabled anyway via cfg.AuthEnabled=false).
func (t *tokenMgr) CountActive(_ context.Context) (int, error) {
	return 0, nil
}

func (t *tokenMgr) Create(_ context.Context, token *repository.AuthToken) error {
	t.s.mu.Lock()
	defer t.s.mu.Unlock()
	t.s.tokens = append(t.s.tokens, token)
	return nil
}

func (t *tokenMgr) List(_ context.Context) ([]repository.AuthToken, error) {
	t.s.mu.Lock()
	defer t.s.mu.Unlock()
	out := make([]repository.AuthToken, len(t.s.tokens))
	for i, x := range t.s.tokens {
		out[i] = *x
	}
	return out, nil
}

func (t *tokenMgr) UpdateLabel(_ context.Context, id uuid.UUID, label string) error {
	t.s.mu.Lock()
	defer t.s.mu.Unlock()
	for _, x := range t.s.tokens {
		if x.ID == id {
			x.Label = label
			return nil
		}
	}
	return repository.ErrNotFound
}

func (t *tokenMgr) Revoke(_ context.Context, id uuid.UUID) error {
	t.s.mu.Lock()
	defer t.s.mu.Unlock()
	for _, x := range t.s.tokens {
		if x.ID == id {
			x.Revoked = true
			return nil
		}
	}
	return repository.ErrNotFound
}

// Compile-time interface checks.
var (
	_ handler.MessageQuerier    = (*messageSvc)(nil)
	_ handler.BufferRater       = (*messageSvc)(nil)
	_ handler.TaskServicer      = (*taskSvc)(nil)
	_ handler.CategoryServicer  = (*categorySvc)(nil)
	_ handler.ScheduleStore     = (*scheduleSvc)(nil)
	_ handler.ScheduleGenerator = noopScheduleGen{}
	_ handler.CalendarFetcher   = noopCalendar{}
	_ handler.RulesManager      = (*rulesSvc)(nil)
	_ handler.ServiceManager    = (*serviceMgr)(nil)
	_ handler.AuthTokenManager  = (*tokenMgr)(nil)
)
