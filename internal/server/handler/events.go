package handler

import (
	"net/http"
	"strconv"
)

// HistoryProvider is the subset of the event hub needed by the events replay
// handler. HistoryJSON returns pre-serialized JSON for all events with
// sequence numbers greater than sinceSeq, suitable for writing directly to the
// HTTP response body.
type HistoryProvider interface {
	HistoryJSON(sinceSeq uint64) ([]byte, error)
}

// EventsHandler returns an http.Handler for GET /api/v1/events?since=<seq>.
// It validates the since query parameter and delegates to the HistoryProvider
// for the actual event history lookup.
func EventsHandler(p HistoryProvider) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := r.URL.Query().Get("since")
		if raw == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid since parameter"}`))
			return
		}

		sinceSeq, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid since parameter"}`))
			return
		}

		data, err := p.HistoryJSON(sinceSeq)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"internal error"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	})
}
