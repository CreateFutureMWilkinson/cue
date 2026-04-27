package handler

import (
	"context"
	"net/http"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/google/uuid"
)

// MessageQuerier is the subset of MessageRepository needed by handlers.
type MessageQuerier interface {
	QueryFiltered(ctx context.Context, filter repository.MessageFilter) ([]*repository.Message, int, error)
	QueryByID(ctx context.Context, id uuid.UUID) (*repository.Message, error)
	Update(ctx context.Context, msg *repository.Message) error
}

// ListNotificationsHandler returns an http.HandlerFunc for GET /api/v1/notifications.
// It queries messages with status "Notified" ordered by created_at descending.
func ListNotificationsHandler(repo MessageQuerier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not implemented", http.StatusNotImplemented)
	}
}
