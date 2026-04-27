package client

import (
	"context"
	"net/http"
	"time"
)

const (
	plannerPath       = "/api/v1/planner"
	plannerActivePath = "/api/v1/planner/active"
)

// ScheduleBlock represents a single time block in a schedule.
//
// Start and End are "HH:MM" strings as emitted by the server. Type is one of
// "focus", "short_break", "long_break", or "meeting". TaskID is nullable; the
// server omits it when a block is not associated with a task. TaskName is
// omitted when empty.
type ScheduleBlock struct {
	Start    string  `json:"start"`
	End      string  `json:"end"`
	Type     string  `json:"type"`
	TaskID   *string `json:"task_id,omitempty"`
	TaskName string  `json:"task_name,omitempty"`
}

// Schedule is the full schedule response returned by /api/v1/planner routes.
//
// Date is "YYYY-MM-DD"; CreatedAt is RFC3339. Blocks is the ordered list of
// time blocks for the day.
type Schedule struct {
	Date      string          `json:"date"`
	Strategy  string          `json:"strategy"`
	Blocks    []ScheduleBlock `json:"blocks"`
	CreatedAt string          `json:"created_at"`
}

// PutScheduleRequest is the PUT body for /api/v1/planner/{date}.
//
// The date comes from the URL path, so it is not part of this body. Callers
// populate Strategy and Blocks; the server rejects unknown block types.
type PutScheduleRequest struct {
	Strategy string          `json:"strategy"`
	Blocks   []ScheduleBlock `json:"blocks"`
}

// ScheduleOption is one candidate schedule returned inside
// GenerateSchedulesResponse.Options. The server emits two options per call
// (e.g. "pomodoro" and "long-form") so the user can pick a strategy.
type ScheduleOption struct {
	Strategy          string          `json:"strategy"`
	TotalFocusMinutes int             `json:"total_focus_minutes"`
	BreakCount        int             `json:"break_count"`
	Blocks            []ScheduleBlock `json:"blocks"`
}

// GenerateSchedulesRequest is the POST body for /api/v1/planner/generate.
//
// Date is optional ("YYYY-MM-DD"). When omitted via the omitempty tag the
// server uses its own target date (planner.TargetDate(time.Now())).
type GenerateSchedulesRequest struct {
	Date string `json:"date,omitempty"`
}

// GenerateSchedulesResponse is the response from /api/v1/planner/generate.
// Options typically contains two strategies the caller can choose between.
type GenerateSchedulesResponse struct {
	Date    string           `json:"date"`
	Options []ScheduleOption `json:"options"`
}

// ScheduleClient wraps /api/v1/planner/* routes: active-day schedule,
// date-addressed schedule CRUD, and generation of candidate schedules.
type ScheduleClient interface {
	ActiveSchedule(ctx context.Context) (*Schedule, error)
	DeleteActiveSchedule(ctx context.Context) error
	GetSchedule(ctx context.Context, date time.Time) (*Schedule, error)
	PutSchedule(ctx context.Context, date time.Time, req PutScheduleRequest) (*Schedule, error)
	DeleteSchedule(ctx context.Context, date time.Time) error
	GenerateSchedules(ctx context.Context, req GenerateSchedulesRequest) (*GenerateSchedulesResponse, error)
}

// scheduleAdapter is the concrete ScheduleClient backed by an *APIClient.
type scheduleAdapter struct {
	client *APIClient
}

// NewScheduleClient returns a ScheduleClient backed by the given APIClient.
func NewScheduleClient(c *APIClient) ScheduleClient {
	return &scheduleAdapter{client: c}
}

// ActiveSchedule issues GET /api/v1/planner/active and returns today's schedule.
func (a *scheduleAdapter) ActiveSchedule(ctx context.Context) (*Schedule, error) {
	var schedule Schedule
	if err := a.client.doJSON(ctx, http.MethodGet, plannerActivePath, nil, &schedule); err != nil {
		return nil, err
	}
	return &schedule, nil
}

// DeleteActiveSchedule issues DELETE /api/v1/planner/active. The server
// responds with 204 No Content; doJSON's nil-out path skips decoding.
func (a *scheduleAdapter) DeleteActiveSchedule(ctx context.Context) error {
	return a.client.doJSON(ctx, http.MethodDelete, plannerActivePath, nil, nil)
}

// GetSchedule issues GET /api/v1/planner/{date} for the given date formatted
// as "YYYY-MM-DD".
func (a *scheduleAdapter) GetSchedule(ctx context.Context, date time.Time) (*Schedule, error) {
	var schedule Schedule
	path := plannerPath + "/" + date.Format("2006-01-02")
	if err := a.client.doJSON(ctx, http.MethodGet, path, nil, &schedule); err != nil {
		return nil, err
	}
	return &schedule, nil
}

// PutSchedule issues PUT /api/v1/planner/{date} with the supplied strategy
// and blocks. The date comes from the URL; req.Strategy and req.Blocks form
// the request body.
func (a *scheduleAdapter) PutSchedule(ctx context.Context, date time.Time, req PutScheduleRequest) (*Schedule, error) {
	var schedule Schedule
	path := plannerPath + "/" + date.Format("2006-01-02")
	if err := a.client.doJSON(ctx, http.MethodPut, path, req, &schedule); err != nil {
		return nil, err
	}
	return &schedule, nil
}

// DeleteSchedule issues DELETE /api/v1/planner/{date}. The server responds
// with 204 No Content; doJSON's nil-out path skips decoding.
func (a *scheduleAdapter) DeleteSchedule(ctx context.Context, date time.Time) error {
	path := plannerPath + "/" + date.Format("2006-01-02")
	return a.client.doJSON(ctx, http.MethodDelete, path, nil, nil)
}

// GenerateSchedules issues POST /api/v1/planner/generate. When req.Date is
// empty the omitempty tag suppresses the field, allowing the server to pick
// its own target date via planner.TargetDate(time.Now()).
func (a *scheduleAdapter) GenerateSchedules(ctx context.Context, req GenerateSchedulesRequest) (*GenerateSchedulesResponse, error) {
	var resp GenerateSchedulesResponse
	if err := a.client.doJSON(ctx, http.MethodPost, plannerPath+"/generate", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
