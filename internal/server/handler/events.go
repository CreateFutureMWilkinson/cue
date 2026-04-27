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
//
// @Summary      Replay buffered events
// @Description  Returns all WebSocket events with sequence numbers greater than
// @Description  `since`. Clients use this after reconnecting to catch up on
// @Description  events missed while disconnected. Payload is a JSON array of
// @Description  EventEnvelope objects matching those pushed over the WebSocket
// @Description  channel; see docs/api/websocket.md for the envelope schema.
// @Tags         events
// @Produce      json
// @Param        since  query     int  true  "Last seq received by the client"
// @Success      200    {array}   object
// @Failure      400    {object}  map[string]string  "invalid since parameter"
// @Failure      500    {object}  map[string]string  "internal error"
// @Router       /api/v1/events [get]
func EventsHandler(p HistoryProvider) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := r.URL.Query().Get("since")
		if raw == "" {
			writeJSONError(w, http.StatusBadRequest, "invalid since parameter")
			return
		}

		sinceSeq, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid since parameter")
			return
		}

		data, err := p.HistoryJSON(sinceSeq)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "internal error")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	})
}
