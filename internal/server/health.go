package server

import (
	"encoding/json"
	"net/http"
)

// SubsystemChecker reports whether a named subsystem is healthy.
type SubsystemChecker interface {
	Name() string
	Check() error
}

// HealthHandler returns an http.HandlerFunc for GET /health that
// responds 200 {"status":"ok"} if the server is running.
func HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// ReadyHandler returns an http.HandlerFunc for GET /health/ready that
// checks all registered subsystems and responds 200 if every check
// returns nil, or 503 if any check returns an error. The response body
// always includes per-subsystem status details.
func ReadyHandler(checkers ...SubsystemChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		subsystems := make(map[string]map[string]string, len(checkers))
		status := http.StatusOK
		for _, c := range checkers {
			entry := map[string]string{"status": "ok"}
			if err := c.Check(); err != nil {
				entry["status"] = "error"
				entry["error"] = err.Error()
				status = http.StatusServiceUnavailable
			}
			subsystems[c.Name()] = entry
		}
		body := map[string]any{
			"status":     statusLabel(status),
			"subsystems": subsystems,
		}
		writeJSON(w, status, body)
	}
}

func statusLabel(code int) string {
	if code == http.StatusOK {
		return "ok"
	}
	return "unavailable"
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
