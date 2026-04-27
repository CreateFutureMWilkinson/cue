package handler

import (
	"context"
	"encoding/json"
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
//
// @Summary      List buffered messages
// @Description  Paginated list of messages awaiting user review (status = Buffered),
// @Description  sorted by created_at descending.
// @Tags         buffer
// @Produce      json
// @Param        limit   query     int  false  "Page size (default 50)"
// @Param        offset  query     int  false  "Page offset (default 0)"
// @Success      200     {object}  handler.bufferedListResponse
// @Failure      500     {object}  map[string]string
// @Router       /api/v1/buffer [get]
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
//
// @Summary      Get buffered message by ID
// @Description  Returns a single buffered message. Responds 404 if the message
// @Description  exists but has a non-Buffered status.
// @Tags         buffer
// @Produce      json
// @Param        id   path      string  true  "Message UUID"
// @Success      200  {object}  handler.messageDetail
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/buffer/{id} [get]
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

// rateRequest is the JSON body for the rate endpoint.
type rateRequest struct {
	Rating   int     `json:"rating"`
	Feedback *string `json:"feedback"`
}

// RateBufferedHandler returns an http.HandlerFunc for POST /api/v1/buffer/{id}/rate.
//
// @Summary      Rate a buffered message
// @Description  Records a 0–10 user rating plus optional free-text feedback and
// @Description  moves the message out of the buffer. Returns 409 if the message
// @Description  has already been resolved.
// @Tags         buffer
// @Accept       json
// @Produce      json
// @Param        id       path      string               true  "Message UUID"
// @Param        request  body      handler.rateRequest  true  "Rating (0–10) and optional feedback"
// @Success      200      {object}  handler.messageDetail
// @Failure      400      {object}  map[string]string
// @Failure      404      {object}  map[string]string
// @Failure      409      {object}  map[string]string  "already resolved"
// @Failure      500      {object}  map[string]string
// @Router       /api/v1/buffer/{id}/rate [post]
func RateBufferedHandler(repo MessageQuerier, buf BufferRater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req rateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.Rating < 0 || req.Rating > 10 {
			writeJSONError(w, http.StatusBadRequest, "rating must be between 0 and 10")
			return
		}

		msg, err := getMessageByPathID(repo, r)
		if err != nil {
			writeNotFoundOrError(w, err)
			return
		}

		if msg.Status != "Buffered" {
			writeJSONError(w, http.StatusConflict, "already resolved")
			return
		}

		if err := buf.SaveRating(r.Context(), msg.ID, req.Rating, req.Feedback); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to save rating")
			return
		}

		updated, err := repo.QueryByID(r.Context(), msg.ID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to fetch updated message")
			return
		}

		writeJSON(w, http.StatusOK, messageToDetail(updated))
	}
}

// DeleteBufferedHandler returns an http.HandlerFunc for DELETE /api/v1/buffer/{id}.
//
// @Summary      Remove a buffered message
// @Description  Deletes the message from the buffer without recording a rating.
// @Description  Returns 409 if the message has already been resolved.
// @Tags         buffer
// @Produce      json
// @Param        id   path      string  true  "Message UUID"
// @Success      200  {object}  handler.messageDetail
// @Failure      404  {object}  map[string]string
// @Failure      409  {object}  map[string]string  "already resolved"
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/buffer/{id} [delete]
func DeleteBufferedHandler(repo MessageQuerier, buf BufferRater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		msg, err := getMessageByPathID(repo, r)
		if err != nil {
			writeNotFoundOrError(w, err)
			return
		}

		if msg.Status != "Buffered" {
			writeJSONError(w, http.StatusConflict, "already resolved")
			return
		}

		if err := buf.DeleteMessage(r.Context(), msg.ID); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to delete message")
			return
		}

		updated, err := repo.QueryByID(r.Context(), msg.ID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to fetch updated message")
			return
		}

		writeJSON(w, http.StatusOK, messageToDetail(updated))
	}
}

// bufferStatsResponse is the JSON envelope for the buffer stats endpoint.
type bufferStatsResponse struct {
	TotalBuffered int            `json:"total_buffered"`
	BySource      map[string]int `json:"by_source"`
}

// BufferStatsHandler returns an http.HandlerFunc for GET /api/v1/buffer/stats.
//
// @Summary      Buffer summary statistics
// @Description  Returns the total count of buffered messages and a breakdown by source.
// @Tags         buffer
// @Produce      json
// @Success      200  {object}  handler.bufferStatsResponse
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/buffer/stats [get]
func BufferStatsHandler(repo MessageQuerier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filter := repository.MessageFilter{
			Status: "Buffered",
			Limit:  10000,
		}

		msgs, total, err := repo.QueryFiltered(r.Context(), filter)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to query buffered messages")
			return
		}

		bySource := make(map[string]int)
		for _, m := range msgs {
			bySource[m.Source]++
		}

		writeJSON(w, http.StatusOK, bufferStatsResponse{
			TotalBuffered: total,
			BySource:      bySource,
		})
	}
}
