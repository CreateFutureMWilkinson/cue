package handler

import "net/http"

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
		http.Error(w, "not implemented", http.StatusNotImplemented)
	})
}
