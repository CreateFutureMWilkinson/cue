package handler

import "net/http"

// ListMessagesHandler returns an http.HandlerFunc for GET /api/v1/messages.
// It supports filtering by status, source, channel, and since (RFC 3339).
func ListMessagesHandler(repo MessageQuerier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not implemented", http.StatusNotImplemented)
	}
}
