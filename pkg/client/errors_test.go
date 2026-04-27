package client_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/pkg/client"
)

// ErrorsSuite covers the APIError type and ParseErrorResponse helper.
type ErrorsSuite struct {
	suite.Suite
}

func TestErrors(t *testing.T) {
	suite.Run(t, new(ErrorsSuite))
}

// TestAPIErrorImplementsErrorInterface verifies *APIError satisfies the error interface.
func (s *ErrorsSuite) TestAPIErrorImplementsErrorInterface() {
	var err error = &client.APIError{StatusCode: 500, Code: client.ErrCodeServerError, Message: "boom"}
	s.NotNil(err, "*APIError should be assignable to error")
}

// TestAPIErrorErrorMessage verifies Error() includes both Code and Message.
func (s *ErrorsSuite) TestAPIErrorErrorMessage() {
	apiErr := &client.APIError{StatusCode: 404, Code: "NOT_FOUND", Message: "missing"}
	msg := apiErr.Error()
	s.Contains(msg, "NOT_FOUND", "Error() should include the Code")
	s.Contains(msg, "missing", "Error() should include the Message")
}

// TestParseErrorResponseFlatBody verifies parsing of the server's flat
// `{"error": "..."}` payload into a populated *APIError.
func (s *ErrorsSuite) TestParseErrorResponseFlatBody() {
	apiErr := client.ParseErrorResponse(404, []byte(`{"error":"not found"}`))
	s.Require().NotNil(apiErr, "ParseErrorResponse should return non-nil for valid body")
	s.Equal(404, apiErr.StatusCode)
	s.Equal(client.ErrCodeNotFound, apiErr.Code)
	s.Equal("not found", apiErr.Message)
}

// TestParseErrorResponseCodeMappingFromStatus verifies the HTTP status -> Code
// mapping for all categories the SDK distinguishes.
func (s *ErrorsSuite) TestParseErrorResponseCodeMappingFromStatus() {
	cases := []struct {
		status int
		code   string
	}{
		{400, client.ErrCodeValidation},
		{401, client.ErrCodeUnauthorized},
		{404, client.ErrCodeNotFound},
		{409, client.ErrCodeConflict},
		{500, client.ErrCodeServerError},
		{502, client.ErrCodeServerError},
		{418, client.ErrCodeClientError},
	}
	for _, tc := range cases {
		body := []byte(`{"error":"x"}`)
		apiErr := client.ParseErrorResponse(tc.status, body)
		s.Require().NotNil(apiErr, "status %d should yield non-nil *APIError", tc.status)
		s.Equal(tc.code, apiErr.Code, "status %d should map to code %q", tc.status, tc.code)
		s.Equal(tc.status, apiErr.StatusCode)
	}
}

// TestParseErrorResponseEmptyBody verifies graceful handling of nil/empty bodies:
// the function still returns a populated *APIError with a default Message.
func (s *ErrorsSuite) TestParseErrorResponseEmptyBody() {
	apiErr := client.ParseErrorResponse(500, nil)
	s.Require().NotNil(apiErr, "ParseErrorResponse should return non-nil for empty body")
	s.Equal(500, apiErr.StatusCode)
	s.Equal(client.ErrCodeServerError, apiErr.Code)
	s.NotEmpty(apiErr.Message, "Message should default to a non-empty string when body is empty")
}

// TestAPIErrorErrorsAs verifies *APIError plays nicely with errors.As when wrapped.
func (s *ErrorsSuite) TestAPIErrorErrorsAs() {
	original := &client.APIError{StatusCode: 409, Code: client.ErrCodeConflict, Message: "duplicate"}
	wrapped := fmt.Errorf("wrap: %w", original)

	var target *client.APIError
	ok := errors.As(wrapped, &target)
	s.True(ok, "errors.As should retrieve *APIError from wrapped error")
	s.Require().NotNil(target)
	s.Equal(409, target.StatusCode)
	s.Equal(client.ErrCodeConflict, target.Code)
	s.Equal("duplicate", target.Message)
}
