package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/google/uuid"
)

// MessageQuerier is the subset of MessageRepository needed by handlers.
type MessageQuerier interface {
	QueryFiltered(ctx context.Context, filter repository.MessageFilter) ([]*repository.Message, int, error)
	QueryByID(ctx context.Context, id uuid.UUID) (*repository.Message, error)
	Update(ctx context.Context, msg *repository.Message) error
}

// notificationItem is the JSON representation of a notification in a list response.
//
// Content is trimmed source-specifically so the UI does not have to drag the full
// message body across the wire: Slack messages are capped at slackContentLimit and
// email messages omit the body entirely — the subject is the visible headline and
// the full body remains available via the detail endpoint.
type notificationItem struct {
	ID              string  `json:"id"`
	Source          string  `json:"source"`
	SourceAccount   string  `json:"source_account"`
	Sender          string  `json:"sender"`
	Channel         string  `json:"channel"`
	Subject         string  `json:"subject"`
	Content         string  `json:"content"`
	WebURL          string  `json:"web_url"`
	ImportanceScore float64 `json:"importance_score"`
	ConfidenceScore float64 `json:"confidence_score"`
	CreatedAt       string  `json:"created_at"`
}

// slackContentLimit caps the Slack notification preview shipped to the UI.
const slackContentLimit = 280

// listResponse is the JSON envelope for paginated list endpoints.
type listResponse struct {
	Notifications []notificationItem `json:"notifications"`
	Total         int                `json:"total"`
}

// ListNotificationsHandler returns an http.HandlerFunc for GET /api/v1/notifications.
// It queries messages with status "Notified" ordered by created_at descending.
//
// @Summary      List active notifications
// @Description  Paginated list of messages currently in the Notified state,
// @Description  sorted by created_at descending.
// @Tags         notifications
// @Produce      json
// @Param        limit   query     int  false  "Page size (default 50)"
// @Param        offset  query     int  false  "Page offset (default 0)"
// @Success      200     {object}  handler.listResponse
// @Failure      500     {object}  map[string]string
// @Router       /api/v1/notifications [get]
func ListNotificationsHandler(repo MessageQuerier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit, offset := parsePagination(r)

		filter := repository.MessageFilter{
			Status: "Notified",
			Limit:  limit,
			Offset: offset,
		}

		msgs, total, err := repo.QueryFiltered(r.Context(), filter)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to query notifications")
			return
		}

		items := make([]notificationItem, len(msgs))
		for i, m := range msgs {
			items[i] = notificationItem{
				ID:              m.ID.String(),
				Source:          m.Source,
				SourceAccount:   m.SourceAccount,
				Sender:          m.Sender,
				Channel:         m.Channel,
				Subject:         m.Subject,
				Content:         trimContentForList(m.Source, m.RawContent),
				WebURL:          m.WebURL,
				ImportanceScore: m.ImportanceScore,
				ConfidenceScore: m.ConfidenceScore,
				CreatedAt:       m.CreatedAt.Format(time.RFC3339),
			}
		}

		writeJSON(w, http.StatusOK, listResponse{
			Notifications: items,
			Total:         total,
		})
	}
}

// trimContentForList returns the wire-payload content for a list-response
// notification item, sized per source. Email omits the body because the
// subject is the visible headline; Slack caps to slackContentLimit so a
// long message does not bloat the response.
func trimContentForList(source, raw string) string {
	switch source {
	case "email":
		return ""
	case "slack":
		if len(raw) <= slackContentLimit {
			return raw
		}
		return raw[:slackContentLimit]
	default:
		return raw
	}
}

// parsePagination extracts limit and offset from query parameters.
func parsePagination(r *http.Request) (int, int) {
	limit := 50
	offset := 0

	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	return limit, offset
}

// writeJSON encodes body as JSON and writes it with the given status code.
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// writeJSONError writes a JSON error response.
func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// messageDetail is the JSON representation of a full message.
type messageDetail struct {
	ID              string  `json:"id"`
	Source          string  `json:"source"`
	SourceAccount   string  `json:"source_account"`
	Channel         string  `json:"channel"`
	Sender          string  `json:"sender"`
	MessageID       string  `json:"message_id"`
	Content         string  `json:"content"`
	ImportanceScore float64 `json:"importance_score"`
	ConfidenceScore float64 `json:"confidence_score"`
	Reasoning       string  `json:"reasoning"`
	Status          string  `json:"status"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
	ResolvedAt      *string `json:"resolved_at,omitempty"`
}

func messageToDetail(m *repository.Message) messageDetail {
	d := messageDetail{
		ID:              m.ID.String(),
		Source:          m.Source,
		SourceAccount:   m.SourceAccount,
		Channel:         m.Channel,
		Sender:          m.Sender,
		MessageID:       m.MessageID,
		Content:         m.RawContent,
		ImportanceScore: m.ImportanceScore,
		ConfidenceScore: m.ConfidenceScore,
		Reasoning:       m.Reasoning,
		Status:          m.Status,
		CreatedAt:       m.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       m.UpdatedAt.Format(time.RFC3339),
	}
	if m.ResolvedAt != nil {
		s := m.ResolvedAt.Format(time.RFC3339)
		d.ResolvedAt = &s
	}
	return d
}

// GetNotificationHandler returns an http.HandlerFunc for GET /api/v1/notifications/{id}.
//
// @Summary      Get notification by ID
// @Tags         notifications
// @Produce      json
// @Param        id   path      string  true  "Notification / message UUID"
// @Success      200  {object}  handler.messageDetail
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/notifications/{id} [get]
func GetNotificationHandler(repo MessageQuerier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		msg, err := getMessageByPathID(repo, r)
		if err != nil {
			writeNotFoundOrError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, messageToDetail(msg))
	}
}

// getMessageByPathID extracts the {id} path param and queries the repo.
func getMessageByPathID(repo MessageQuerier, r *http.Request) (*repository.Message, error) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, repository.ErrNotFound
	}
	return repo.QueryByID(r.Context(), id)
}

// writeNotFoundOrError writes 404 for ErrNotFound, 500 otherwise.
func writeNotFoundOrError(w http.ResponseWriter, err error) {
	if errors.Is(err, repository.ErrNotFound) {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSONError(w, http.StatusInternalServerError, "internal error")
}

// ResolveNotificationHandler returns an http.HandlerFunc for POST /api/v1/notifications/{id}/resolve.
//
// @Summary      Resolve a notification
// @Description  Marks the notification as resolved and sets resolved_at to now.
// @Description  Returns 409 if the notification is already resolved.
// @Tags         notifications
// @Produce      json
// @Param        id   path      string  true  "Notification UUID"
// @Success      200  {object}  handler.messageDetail
// @Failure      404  {object}  map[string]string
// @Failure      409  {object}  map[string]string  "already resolved"
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/notifications/{id}/resolve [post]
func ResolveNotificationHandler(repo MessageQuerier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		msg, err := getMessageByPathID(repo, r)
		if err != nil {
			writeNotFoundOrError(w, err)
			return
		}
		if msg.Status == "Resolved" {
			writeJSONError(w, http.StatusConflict, "already resolved")
			return
		}
		now := time.Now().UTC()
		msg.Status = "Resolved"
		msg.ResolvedAt = &now
		msg.UpdatedAt = now
		if err := repo.Update(r.Context(), msg); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to update")
			return
		}
		writeJSON(w, http.StatusOK, messageToDetail(msg))
	}
}

// DismissNotificationHandler returns an http.HandlerFunc for POST /api/v1/notifications/{id}/dismiss.
//
// @Summary      Dismiss a notification
// @Description  Marks the notification as Ignored. Unlike resolve, no resolved_at timestamp is set.
// @Tags         notifications
// @Produce      json
// @Param        id   path      string  true  "Notification UUID"
// @Success      200  {object}  handler.messageDetail
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/notifications/{id}/dismiss [post]
func DismissNotificationHandler(repo MessageQuerier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		msg, err := getMessageByPathID(repo, r)
		if err != nil {
			writeNotFoundOrError(w, err)
			return
		}
		msg.Status = "Ignored"
		msg.UpdatedAt = time.Now().UTC()
		if err := repo.Update(r.Context(), msg); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to update")
			return
		}
		writeJSON(w, http.StatusOK, messageToDetail(msg))
	}
}
