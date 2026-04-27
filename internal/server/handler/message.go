package handler

import (
	"net/http"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
)

// messageListItem is the JSON representation of a message in a list response.
type messageListItem struct {
	ID              string  `json:"id"`
	Source          string  `json:"source"`
	SourceAccount   string  `json:"source_account"`
	Sender          string  `json:"sender"`
	Channel         string  `json:"channel"`
	Content         string  `json:"content"`
	ImportanceScore float64 `json:"importance_score"`
	ConfidenceScore float64 `json:"confidence_score"`
	Status          string  `json:"status"`
	CreatedAt       string  `json:"created_at"`
}

// messagesListResponse is the JSON envelope for the messages list endpoint.
type messagesListResponse struct {
	Messages []messageListItem `json:"messages"`
	Total    int               `json:"total"`
}

// ListMessagesHandler returns an http.HandlerFunc for GET /api/v1/messages.
// It supports filtering by status, source, channel, and since (RFC 3339).
func ListMessagesHandler(repo MessageQuerier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit, offset := parsePagination(r)
		q := r.URL.Query()

		filter := repository.MessageFilter{
			Status:  q.Get("status"),
			Source:  q.Get("source"),
			Channel: q.Get("channel"),
			Limit:   limit,
			Offset:  offset,
		}

		if since := q.Get("since"); since != "" {
			t, err := time.Parse(time.RFC3339, since)
			if err == nil {
				filter.Since = &t
			}
		}

		msgs, total, err := repo.QueryFiltered(r.Context(), filter)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to query messages")
			return
		}

		items := make([]messageListItem, len(msgs))
		for i, m := range msgs {
			items[i] = messageListItem{
				ID:              m.ID.String(),
				Source:          m.Source,
				SourceAccount:   m.SourceAccount,
				Sender:          m.Sender,
				Channel:         m.Channel,
				Content:         m.RawContent,
				ImportanceScore: m.ImportanceScore,
				ConfidenceScore: m.ConfidenceScore,
				Status:          m.Status,
				CreatedAt:       m.CreatedAt.Format(time.RFC3339),
			}
		}

		writeJSON(w, http.StatusOK, messagesListResponse{
			Messages: items,
			Total:    total,
		})
	}
}

// GetMessageHandler returns an http.HandlerFunc for GET /api/v1/messages/{id}.
func GetMessageHandler(repo MessageQuerier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		msg, err := getMessageByPathID(repo, r)
		if err != nil {
			writeNotFoundOrError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, messageToDetail(msg))
	}
}
