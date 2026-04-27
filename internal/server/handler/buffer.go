package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/google/uuid"
)

// BufferRater is the subset of buffer.BufferService needed for rating/dismissing.
type BufferRater interface {
	SaveRating(ctx context.Context, messageID uuid.UUID, rating int, feedback *string) error
	DeleteMessage(ctx context.Context, messageID uuid.UUID) error
}

// bufferedMessageItem is the JSON representation of a buffered message in a list response.
type bufferedMessageItem struct {
	ID              string  `json:"id"`
	Source          string  `json:"source"`
	SourceAccount   string  `json:"source_account"`
	Sender          string  `json:"sender"`
	Channel         string  `json:"channel"`
	Content         string  `json:"content"`
	ImportanceScore float64 `json:"importance_score"`
	ConfidenceScore float64 `json:"confidence_score"`
	Reasoning       string  `json:"reasoning"`
	CreatedAt       string  `json:"created_at"`
}

// bufferedListResponse is the JSON envelope for the buffer list endpoint.
type bufferedListResponse struct {
	Messages []bufferedMessageItem `json:"messages"`
	Total    int                   `json:"total"`
	Count    int                   `json:"count"`
}

func messageToBufferedItem(m *repository.Message) bufferedMessageItem {
	return bufferedMessageItem{
		ID:              m.ID.String(),
		Source:          m.Source,
		SourceAccount:   m.SourceAccount,
		Sender:          m.Sender,
		Channel:         m.Channel,
		Content:         m.RawContent,
		ImportanceScore: m.ImportanceScore,
		ConfidenceScore: m.ConfidenceScore,
		Reasoning:       m.Reasoning,
		CreatedAt:       m.CreatedAt.Format(time.RFC3339),
	}
}

// ListBufferedHandler returns an http.HandlerFunc for GET /api/v1/buffer.
// It queries messages with status "Buffered" ordered by created_at descending.
func ListBufferedHandler(repo MessageQuerier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit, offset := parsePagination(r)

		filter := repository.MessageFilter{
			Status: "Buffered",
			Limit:  limit,
			Offset: offset,
		}

		msgs, total, err := repo.QueryFiltered(r.Context(), filter)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to query buffered messages")
			return
		}

		items := make([]bufferedMessageItem, len(msgs))
		for i, m := range msgs {
			items[i] = messageToBufferedItem(m)
		}

		writeJSON(w, http.StatusOK, bufferedListResponse{
			Messages: items,
			Total:    total,
			Count:    len(items),
		})
	}
}

// GetBufferedHandler returns an http.HandlerFunc for GET /api/v1/buffer/{id}.
func GetBufferedHandler(repo MessageQuerier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		msg, err := getMessageByPathID(repo, r)
		if err != nil {
			writeNotFoundOrError(w, err)
			return
		}
		if msg.Status != "Buffered" {
			writeJSONError(w, http.StatusNotFound, "not found")
			return
		}
		writeJSON(w, http.StatusOK, messageToDetail(msg))
	}
}

// RateBufferedHandler returns an http.HandlerFunc for POST /api/v1/buffer/{id}/rate.
func RateBufferedHandler(repo MessageQuerier, buf BufferRater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSONError(w, http.StatusNotImplemented, "not implemented")
	}
}

// DeleteBufferedHandler returns an http.HandlerFunc for DELETE /api/v1/buffer/{id}.
func DeleteBufferedHandler(repo MessageQuerier, buf BufferRater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSONError(w, http.StatusNotImplemented, "not implemented")
	}
}

// BufferStatsHandler returns an http.HandlerFunc for GET /api/v1/buffer/stats.
func BufferStatsHandler(repo MessageQuerier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSONError(w, http.StatusNotImplemented, "not implemented")
	}
}
