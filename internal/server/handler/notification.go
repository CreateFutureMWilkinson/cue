package handler

import (
	"context"
	"encoding/json"
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
type notificationItem struct {
	ID              string  `json:"id"`
	Source          string  `json:"source"`
	SourceAccount   string  `json:"source_account"`
	Sender          string  `json:"sender"`
	Channel         string  `json:"channel"`
	Content         string  `json:"content"`
	ImportanceScore float64 `json:"importance_score"`
	ConfidenceScore float64 `json:"confidence_score"`
	CreatedAt       string  `json:"created_at"`
}

// listResponse is the JSON envelope for paginated list endpoints.
type listResponse struct {
	Notifications []notificationItem `json:"notifications"`
	Total         int                `json:"total"`
}

// ListNotificationsHandler returns an http.HandlerFunc for GET /api/v1/notifications.
// It queries messages with status "Notified" ordered by created_at descending.
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
				Content:         m.RawContent,
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
