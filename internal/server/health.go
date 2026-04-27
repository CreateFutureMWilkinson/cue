package server

import "net/http"

// SubsystemChecker reports whether a named subsystem is healthy.
type SubsystemChecker interface {
	Name() string
	Check() error
}

// HealthHandler returns an http.HandlerFunc for GET /health that
// responds 200 {"status":"ok"} if the server is running.
func HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// stub: returns nothing
	}
}

// ReadyHandler returns an http.HandlerFunc for GET /health/ready that
// checks all registered subsystems and responds 200 or 503.
func ReadyHandler(checkers ...SubsystemChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// stub: returns nothing
	}
}
