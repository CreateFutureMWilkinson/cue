package client

import (
	"fmt"
)

// Error code constants derived from HTTP status codes in server error responses.
const (
	ErrCodeValidation   = "VALIDATION_ERROR"
	ErrCodeUnauthorized = "UNAUTHORIZED"
	ErrCodeNotFound     = "NOT_FOUND"
	ErrCodeConflict     = "CONFLICT"
	ErrCodeClientError  = "CLIENT_ERROR"
	ErrCodeServerError  = "SERVER_ERROR"
	ErrCodeTokenIssued  = "TOKEN_ISSUED"
)

// APIError represents a structured error response from the cue-server API.
//
// The server emits flat `{"error": "message"}` JSON payloads. APIError
// augments that with the HTTP status code and a synthesized Code constant
// (see Err* constants) so SDK consumers can branch on error category
// without parsing the message string.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
}

// Error implements the error interface.
func (e *APIError) Error() string {
	return fmt.Sprintf("not implemented")
}

// ParseErrorResponse parses a server error response body into an APIError.
//
// The server emits flat `{"error": "message"}` payloads; this function reads
// the message and derives the Code from the HTTP status. Exported so SDK
// consumers can reuse the parsing logic when they receive a raw response
// from a custom transport.
func ParseErrorResponse(statusCode int, body []byte) *APIError {
	return nil
}
