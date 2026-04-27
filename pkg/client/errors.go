package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
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
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// ParseErrorResponse parses a server error response body into an APIError.
//
// The server emits flat `{"error": "message"}` payloads; this function reads
// the message and derives the Code from the HTTP status. Exported so SDK
// consumers can reuse the parsing logic when they receive a raw response
// from a custom transport.
func ParseErrorResponse(statusCode int, body []byte) *APIError {
	code := codeForStatus(statusCode)
	message := messageFromBody(body, statusCode)
	return &APIError{
		StatusCode: statusCode,
		Code:       code,
		Message:    message,
	}
}

// codeForStatus maps an HTTP status code to one of the SDK's Err* constants.
func codeForStatus(statusCode int) string {
	switch statusCode {
	case http.StatusBadRequest:
		return ErrCodeValidation
	case http.StatusUnauthorized:
		return ErrCodeUnauthorized
	case http.StatusNotFound:
		return ErrCodeNotFound
	case http.StatusConflict:
		return ErrCodeConflict
	}
	switch {
	case statusCode >= 500 && statusCode <= 599:
		return ErrCodeServerError
	case statusCode >= 400 && statusCode <= 499:
		return ErrCodeClientError
	default:
		// Conservative default for unexpected status codes.
		return ErrCodeServerError
	}
}

// messageFromBody attempts to extract the server's flat `{"error": "..."}`
// message from body. Falls back to http.StatusText (or "HTTP <code>") on
// empty or unparseable input.
func messageFromBody(body []byte, statusCode int) string {
	if len(body) > 0 {
		var payload struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(body, &payload); err == nil && payload.Error != "" {
			return payload.Error
		}
	}
	if text := http.StatusText(statusCode); text != "" {
		return text
	}
	return "HTTP " + strconv.Itoa(statusCode)
}
