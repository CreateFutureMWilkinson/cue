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
//
// @Summary      Liveness check
// @Description  Returns 200 {"status":"ok"} whenever the HTTP server is running.
// @Description  Does not inspect any subsystem; use /health/ready for readiness.
// @Tags         health
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /health [get]
// @Router       /api/v1/health [get]
func HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// ReadyHandler returns an http.HandlerFunc for GET /health/ready that
// checks all registered subsystems and responds 200 if every check
// returns nil, or 503 if any check returns an error. The response body
// always includes per-subsystem status details.
//
// @Summary      Readiness check
// @Description  Runs every registered subsystem check and returns 200 only if
// @Description  all report healthy. Returns 503 with per-subsystem error detail
// @Description  when any check fails. The response body always includes a
// @Description  "subsystems" object keyed by subsystem name.
// @Tags         health
// @Produce      json
// @Success      200  {object}  map[string]any
// @Failure      503  {object}  map[string]any
// @Router       /health/ready [get]
// @Router       /api/v1/health/ready [get]
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
