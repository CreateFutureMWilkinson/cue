package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/service/servicemanager"
	"github.com/google/uuid"
)

// ServiceManager is the subset of servicemanager.ServiceManager needed by service handlers.
type ServiceManager interface {
	ListSlackAccounts(ctx context.Context) ([]*repository.SlackAccount, error)
	ListEmailAccounts(ctx context.Context) ([]*repository.EmailAccount, error)
	ListCalendarAccounts(ctx context.Context) ([]*repository.CalendarAccount, error)
	GetSlackAccount(ctx context.Context, id uuid.UUID) (*repository.SlackAccount, error)
	GetEmailAccount(ctx context.Context, id uuid.UUID) (*repository.EmailAccount, error)
	GetCalendarAccount(ctx context.Context, id uuid.UUID) (*repository.CalendarAccount, error)
	CreateSlackAccount(ctx context.Context, acct *repository.SlackAccount) (*repository.SlackAccount, error)
	CreateEmailAccount(ctx context.Context, acct *repository.EmailAccount) (*repository.EmailAccount, error)
	CreateCalendarAccount(ctx context.Context, acct *repository.CalendarAccount) (*repository.CalendarAccount, error)
	UpdateSlackAccount(ctx context.Context, id uuid.UUID, acct *repository.SlackAccount) (*repository.SlackAccount, error)
	UpdateEmailAccount(ctx context.Context, id uuid.UUID, acct *repository.EmailAccount) (*repository.EmailAccount, error)
	UpdateCalendarAccount(ctx context.Context, id uuid.UUID, acct *repository.CalendarAccount) (*repository.CalendarAccount, error)
	DeleteSlackAccount(ctx context.Context, id uuid.UUID) error
	DeleteEmailAccount(ctx context.Context, id uuid.UUID) error
	DeleteCalendarAccount(ctx context.Context, id uuid.UUID) error
	ToggleSlackAccount(ctx context.Context, id uuid.UUID, enabled bool) error
	ToggleEmailAccount(ctx context.Context, id uuid.UUID, enabled bool) error
	ToggleCalendarAccount(ctx context.Context, id uuid.UUID, enabled bool) error
	Status(ctx context.Context) ([]servicemanager.ServiceStatus, error)
}

// --- JSON response types ---

type slackAccountItem struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	WorkspaceID         string `json:"workspace_id"`
	Username            string `json:"username"`
	WebURL              string `json:"web_url"`
	PollIntervalSeconds int    `json:"poll_interval_seconds"`
	Enabled             bool   `json:"enabled"`
	CreatedAt           string `json:"created_at"`
}

type emailAccountItem struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	IMAPHost            string `json:"imap_host"`
	IMAPPort            int    `json:"imap_port"`
	Username            string `json:"username"`
	Encryption          string `json:"encryption"`
	WebURL              string `json:"web_url"`
	PollIntervalSeconds int    `json:"poll_interval_seconds"`
	Enabled             bool   `json:"enabled"`
	CreatedAt           string `json:"created_at"`
}

type calendarAccountItem struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	ICSURL              string `json:"ics_url"`
	PollIntervalSeconds int    `json:"poll_interval_seconds"`
	Enabled             bool   `json:"enabled"`
	CreatedAt           string `json:"created_at"`
}

// --- JSON request types ---

type createSlackRequest struct {
	Name                string `json:"name"`
	BotToken            string `json:"bot_token"`
	WorkspaceID         string `json:"workspace_id"`
	Username            string `json:"username"`
	WebURL              string `json:"web_url"`
	PollIntervalSeconds int    `json:"poll_interval_seconds"`
	Enabled             bool   `json:"enabled"`
}

type createEmailRequest struct {
	Name                string `json:"name"`
	IMAPHost            string `json:"imap_host"`
	IMAPPort            int    `json:"imap_port"`
	Username            string `json:"username"`
	Password            string `json:"password"`
	Encryption          string `json:"encryption"`
	WebURL              string `json:"web_url"`
	PollIntervalSeconds int    `json:"poll_interval_seconds"`
	Enabled             bool   `json:"enabled"`
}

type createCalendarRequest struct {
	Name                string `json:"name"`
	ICSURL              string `json:"ics_url"`
	PollIntervalSeconds int    `json:"poll_interval_seconds"`
	Enabled             bool   `json:"enabled"`
}

type toggleRequest struct {
	Enabled bool `json:"enabled"`
}

type serviceStatusItem struct {
	ID                string `json:"id"`
	Type              string `json:"type"`
	Name              string `json:"name"`
	Enabled           bool   `json:"enabled"`
	WatcherRegistered bool   `json:"watcher_registered"`
}

// --- Conversion helpers ---

func slackToItem(a *repository.SlackAccount) slackAccountItem {
	return slackAccountItem{
		ID:                  a.ID.String(),
		Name:                a.FriendlyName,
		WorkspaceID:         a.WorkspaceID,
		Username:            a.Username,
		WebURL:              a.WebURL,
		PollIntervalSeconds: a.PollIntervalSeconds,
		Enabled:             a.Enabled,
		CreatedAt:           a.CreatedAt.Format(time.RFC3339),
	}
}

func emailToItem(a *repository.EmailAccount) emailAccountItem {
	return emailAccountItem{
		ID:                  a.ID.String(),
		Name:                a.FriendlyName,
		IMAPHost:            a.IMAPHost,
		IMAPPort:            a.IMAPPort,
		Username:            a.Username,
		Encryption:          a.Encryption,
		WebURL:              a.WebURL,
		PollIntervalSeconds: a.PollIntervalSeconds,
		Enabled:             a.Enabled,
		CreatedAt:           a.CreatedAt.Format(time.RFC3339),
	}
}

func calendarToItem(a *repository.CalendarAccount) calendarAccountItem {
	return calendarAccountItem{
		ID:                  a.ID.String(),
		Name:                a.Name,
		ICSURL:              a.ICSURL,
		PollIntervalSeconds: a.PollIntervalSeconds,
		Enabled:             a.Enabled,
		CreatedAt:           a.CreatedAt.Format(time.RFC3339),
	}
}

// --- Helpers ---

// parseServiceID extracts and parses the {id} path parameter as a UUID.
func parseServiceID(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(r.PathValue("id"))
}

// isNotFound checks whether an error represents a not-found condition.
func isNotFound(err error) bool {
	return strings.Contains(err.Error(), "not found")
}

// writeServiceError writes 400 for bad IDs, 404 for not-found, 500 otherwise.
func writeServiceError(w http.ResponseWriter, err error) {
	if isNotFound(err) {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSONError(w, http.StatusInternalServerError, "internal error")
}

// --- Slack handlers ---

// ListSlackAccountsHandler returns an http.HandlerFunc for GET /api/v1/services/slack.
//
// @Summary      List Slack accounts
// @Tags         services-slack
// @Produce      json
// @Success      200  {object}  map[string]any
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/services/slack [get]
func ListSlackAccountsHandler(svc ServiceManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accounts, err := svc.ListSlackAccounts(r.Context())
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to list slack accounts")
			return
		}

		items := make([]slackAccountItem, len(accounts))
		for i, a := range accounts {
			items[i] = slackToItem(a)
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"accounts": items,
			"count":    len(items),
		})
	}
}

// GetSlackAccountHandler returns an http.HandlerFunc for GET /api/v1/services/slack/{id}.
//
// @Summary      Get Slack account
// @Tags         services-slack
// @Produce      json
// @Param        id   path      string  true  "Account UUID"
// @Success      200  {object}  handler.slackAccountItem
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/services/slack/{id} [get]
func GetSlackAccountHandler(svc ServiceManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseServiceID(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}

		acct, err := svc.GetSlackAccount(r.Context(), id)
		if err != nil {
			writeServiceError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, slackToItem(acct))
	}
}

// CreateSlackAccountHandler returns an http.HandlerFunc for POST /api/v1/services/slack.
//
// @Summary      Create Slack account
// @Tags         services-slack
// @Accept       json
// @Produce      json
// @Param        request  body      handler.createSlackRequest  true  "Slack account fields"
// @Success      201      {object}  handler.slackAccountItem
// @Failure      400      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /api/v1/services/slack [post]
func CreateSlackAccountHandler(svc ServiceManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createSlackRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		acct := &repository.SlackAccount{
			FriendlyName:        req.Name,
			Token:               req.BotToken,
			WorkspaceID:         req.WorkspaceID,
			Username:            req.Username,
			WebURL:              req.WebURL,
			PollIntervalSeconds: req.PollIntervalSeconds,
			Enabled:             req.Enabled,
		}

		created, err := svc.CreateSlackAccount(r.Context(), acct)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to create slack account")
			return
		}

		writeJSON(w, http.StatusCreated, slackToItem(created))
	}
}

// UpdateSlackAccountHandler returns an http.HandlerFunc for PUT /api/v1/services/slack/{id}.
//
// @Summary      Update Slack account
// @Tags         services-slack
// @Accept       json
// @Produce      json
// @Param        id       path      string                      true  "Account UUID"
// @Param        request  body      handler.createSlackRequest  true  "Slack account fields"
// @Success      200      {object}  handler.slackAccountItem
// @Failure      400      {object}  map[string]string
// @Failure      404      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /api/v1/services/slack/{id} [put]
func UpdateSlackAccountHandler(svc ServiceManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseServiceID(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}

		var req createSlackRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		acct := &repository.SlackAccount{
			FriendlyName:        req.Name,
			Token:               req.BotToken,
			WorkspaceID:         req.WorkspaceID,
			Username:            req.Username,
			WebURL:              req.WebURL,
			PollIntervalSeconds: req.PollIntervalSeconds,
			Enabled:             req.Enabled,
		}

		updated, err := svc.UpdateSlackAccount(r.Context(), id, acct)
		if err != nil {
			writeServiceError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, slackToItem(updated))
	}
}

// DeleteSlackAccountHandler returns an http.HandlerFunc for DELETE /api/v1/services/slack/{id}.
//
// @Summary      Delete Slack account
// @Tags         services-slack
// @Param        id   path  string  true  "Account UUID"
// @Success      204  "No Content"
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/services/slack/{id} [delete]
func DeleteSlackAccountHandler(svc ServiceManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseServiceID(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}

		if err := svc.DeleteSlackAccount(r.Context(), id); err != nil {
			writeServiceError(w, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// ToggleSlackAccountHandler returns an http.HandlerFunc for POST /api/v1/services/slack/{id}/toggle.
//
// @Summary      Enable or disable Slack account
// @Tags         services-slack
// @Accept       json
// @Param        id       path  string                  true  "Account UUID"
// @Param        request  body  handler.toggleRequest   true  "Desired enabled state"
// @Success      204  "No Content"
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/services/slack/{id}/toggle [post]
func ToggleSlackAccountHandler(svc ServiceManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseServiceID(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}

		var req toggleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if err := svc.ToggleSlackAccount(r.Context(), id, req.Enabled); err != nil {
			writeServiceError(w, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// --- Email handlers ---

// ListEmailAccountsHandler returns an http.HandlerFunc for GET /api/v1/services/email.
//
// @Summary      List email accounts
// @Tags         services-email
// @Produce      json
// @Success      200  {object}  map[string]any
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/services/email [get]
func ListEmailAccountsHandler(svc ServiceManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accounts, err := svc.ListEmailAccounts(r.Context())
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to list email accounts")
			return
		}

		items := make([]emailAccountItem, len(accounts))
		for i, a := range accounts {
			items[i] = emailToItem(a)
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"accounts": items,
			"count":    len(items),
		})
	}
}

// GetEmailAccountHandler returns an http.HandlerFunc for GET /api/v1/services/email/{id}.
//
// @Summary      Get email account
// @Tags         services-email
// @Produce      json
// @Param        id   path      string  true  "Account UUID"
// @Success      200  {object}  handler.emailAccountItem
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/services/email/{id} [get]
func GetEmailAccountHandler(svc ServiceManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseServiceID(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}

		acct, err := svc.GetEmailAccount(r.Context(), id)
		if err != nil {
			writeServiceError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, emailToItem(acct))
	}
}

// CreateEmailAccountHandler returns an http.HandlerFunc for POST /api/v1/services/email.
//
// @Summary      Create email account
// @Tags         services-email
// @Accept       json
// @Produce      json
// @Param        request  body      handler.createEmailRequest  true  "Email account fields"
// @Success      201      {object}  handler.emailAccountItem
// @Failure      400      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /api/v1/services/email [post]
func CreateEmailAccountHandler(svc ServiceManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createEmailRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		acct := &repository.EmailAccount{
			FriendlyName:        req.Name,
			IMAPHost:            req.IMAPHost,
			IMAPPort:            req.IMAPPort,
			Username:            req.Username,
			Password:            req.Password,
			Encryption:          req.Encryption,
			WebURL:              req.WebURL,
			PollIntervalSeconds: req.PollIntervalSeconds,
			Enabled:             req.Enabled,
		}

		created, err := svc.CreateEmailAccount(r.Context(), acct)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to create email account")
			return
		}

		writeJSON(w, http.StatusCreated, emailToItem(created))
	}
}

// UpdateEmailAccountHandler returns an http.HandlerFunc for PUT /api/v1/services/email/{id}.
//
// @Summary      Update email account
// @Tags         services-email
// @Accept       json
// @Produce      json
// @Param        id       path      string                      true  "Account UUID"
// @Param        request  body      handler.createEmailRequest  true  "Email account fields"
// @Success      200      {object}  handler.emailAccountItem
// @Failure      400      {object}  map[string]string
// @Failure      404      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /api/v1/services/email/{id} [put]
func UpdateEmailAccountHandler(svc ServiceManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseServiceID(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}

		var req createEmailRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		acct := &repository.EmailAccount{
			FriendlyName:        req.Name,
			IMAPHost:            req.IMAPHost,
			IMAPPort:            req.IMAPPort,
			Username:            req.Username,
			Password:            req.Password,
			Encryption:          req.Encryption,
			WebURL:              req.WebURL,
			PollIntervalSeconds: req.PollIntervalSeconds,
			Enabled:             req.Enabled,
		}

		updated, err := svc.UpdateEmailAccount(r.Context(), id, acct)
		if err != nil {
			writeServiceError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, emailToItem(updated))
	}
}

// DeleteEmailAccountHandler returns an http.HandlerFunc for DELETE /api/v1/services/email/{id}.
//
// @Summary      Delete email account
// @Tags         services-email
// @Param        id   path  string  true  "Account UUID"
// @Success      204  "No Content"
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/services/email/{id} [delete]
func DeleteEmailAccountHandler(svc ServiceManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseServiceID(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}

		if err := svc.DeleteEmailAccount(r.Context(), id); err != nil {
			writeServiceError(w, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// ToggleEmailAccountHandler returns an http.HandlerFunc for POST /api/v1/services/email/{id}/toggle.
//
// @Summary      Enable or disable email account
// @Tags         services-email
// @Accept       json
// @Param        id       path  string                  true  "Account UUID"
// @Param        request  body  handler.toggleRequest   true  "Desired enabled state"
// @Success      204  "No Content"
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/services/email/{id}/toggle [post]
func ToggleEmailAccountHandler(svc ServiceManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseServiceID(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}

		var req toggleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if err := svc.ToggleEmailAccount(r.Context(), id, req.Enabled); err != nil {
			writeServiceError(w, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// --- Calendar handlers ---

// ListCalendarAccountsHandler returns an http.HandlerFunc for GET /api/v1/services/calendar.
//
// @Summary      List calendar accounts
// @Tags         services-calendar
// @Produce      json
// @Success      200  {object}  map[string]any
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/services/calendar [get]
func ListCalendarAccountsHandler(svc ServiceManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accounts, err := svc.ListCalendarAccounts(r.Context())
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to list calendar accounts")
			return
		}

		items := make([]calendarAccountItem, len(accounts))
		for i, a := range accounts {
			items[i] = calendarToItem(a)
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"accounts": items,
			"count":    len(items),
		})
	}
}

// GetCalendarAccountHandler returns an http.HandlerFunc for GET /api/v1/services/calendar/{id}.
//
// @Summary      Get calendar account
// @Tags         services-calendar
// @Produce      json
// @Param        id   path      string  true  "Account UUID"
// @Success      200  {object}  handler.calendarAccountItem
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/services/calendar/{id} [get]
func GetCalendarAccountHandler(svc ServiceManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseServiceID(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}

		acct, err := svc.GetCalendarAccount(r.Context(), id)
		if err != nil {
			writeServiceError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, calendarToItem(acct))
	}
}

// CreateCalendarAccountHandler returns an http.HandlerFunc for POST /api/v1/services/calendar.
//
// @Summary      Create calendar account
// @Tags         services-calendar
// @Accept       json
// @Produce      json
// @Param        request  body      handler.createCalendarRequest  true  "Calendar account fields"
// @Success      201      {object}  handler.calendarAccountItem
// @Failure      400      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /api/v1/services/calendar [post]
func CreateCalendarAccountHandler(svc ServiceManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createCalendarRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		acct := &repository.CalendarAccount{
			Name:                req.Name,
			ICSURL:              req.ICSURL,
			PollIntervalSeconds: req.PollIntervalSeconds,
			Enabled:             req.Enabled,
		}

		created, err := svc.CreateCalendarAccount(r.Context(), acct)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to create calendar account")
			return
		}

		writeJSON(w, http.StatusCreated, calendarToItem(created))
	}
}

// UpdateCalendarAccountHandler returns an http.HandlerFunc for PUT /api/v1/services/calendar/{id}.
//
// @Summary      Update calendar account
// @Tags         services-calendar
// @Accept       json
// @Produce      json
// @Param        id       path      string                         true  "Account UUID"
// @Param        request  body      handler.createCalendarRequest  true  "Calendar account fields"
// @Success      200      {object}  handler.calendarAccountItem
// @Failure      400      {object}  map[string]string
// @Failure      404      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /api/v1/services/calendar/{id} [put]
func UpdateCalendarAccountHandler(svc ServiceManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseServiceID(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}

		var req createCalendarRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		acct := &repository.CalendarAccount{
			Name:                req.Name,
			ICSURL:              req.ICSURL,
			PollIntervalSeconds: req.PollIntervalSeconds,
			Enabled:             req.Enabled,
		}

		updated, err := svc.UpdateCalendarAccount(r.Context(), id, acct)
		if err != nil {
			writeServiceError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, calendarToItem(updated))
	}
}

// DeleteCalendarAccountHandler returns an http.HandlerFunc for DELETE /api/v1/services/calendar/{id}.
//
// @Summary      Delete calendar account
// @Tags         services-calendar
// @Param        id   path  string  true  "Account UUID"
// @Success      204  "No Content"
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/services/calendar/{id} [delete]
func DeleteCalendarAccountHandler(svc ServiceManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseServiceID(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}

		if err := svc.DeleteCalendarAccount(r.Context(), id); err != nil {
			writeServiceError(w, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// ToggleCalendarAccountHandler returns an http.HandlerFunc for POST /api/v1/services/calendar/{id}/toggle.
//
// @Summary      Enable or disable calendar account
// @Tags         services-calendar
// @Accept       json
// @Param        id       path  string                  true  "Account UUID"
// @Param        request  body  handler.toggleRequest   true  "Desired enabled state"
// @Success      204  "No Content"
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/services/calendar/{id}/toggle [post]
func ToggleCalendarAccountHandler(svc ServiceManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseServiceID(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}

		var req toggleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if err := svc.ToggleCalendarAccount(r.Context(), id, req.Enabled); err != nil {
			writeServiceError(w, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// --- Status handler ---

// ServiceStatusHandler returns an http.HandlerFunc for GET /api/v1/services/status.
//
// @Summary      Service status overview
// @Description  Returns every configured service (Slack, Email, Calendar) with
// @Description  its enabled flag and whether an orchestrator watcher is running.
// @Tags         services
// @Produce      json
// @Success      200  {object}  map[string]any
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/services/status [get]
func ServiceStatusHandler(svc ServiceManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		statuses, err := svc.Status(r.Context())
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to get service status")
			return
		}

		items := make([]serviceStatusItem, len(statuses))
		for i, s := range statuses {
			items[i] = serviceStatusItem{
				ID:                s.ID.String(),
				Type:              s.Type,
				Name:              s.Name,
				Enabled:           s.Enabled,
				WatcherRegistered: s.WatcherRegistered,
			}
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"services": items,
			"count":    len(items),
		})
	}
}
