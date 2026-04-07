package validation_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/service/validation"
)

type SlackValidatorSuite struct {
	suite.Suite
}

func TestSlackValidator(t *testing.T) {
	suite.Run(t, new(SlackValidatorSuite))
}

func (s *SlackValidatorSuite) TestSlackValidator_InvalidToken() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal("/auth.test", r.URL.Path)
		s.Equal("Bearer bad-token", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":false,"error":"invalid_auth"}`))
	}))
	defer server.Close()

	v := validation.NewSlackAPIValidator(validation.WithSlackBaseURL(server.URL))
	err := v.ValidateSlack(context.Background(), "bad-token")

	s.Error(err)
	s.Contains(err.Error(), "invalid_auth")
}

func (s *SlackValidatorSuite) TestSlackValidator_ValidToken() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal("/auth.test", r.URL.Path)
		s.Equal("Bearer xoxb-valid", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true,"user_id":"U123","team":"Test"}`))
	}))
	defer server.Close()

	v := validation.NewSlackAPIValidator(validation.WithSlackBaseURL(server.URL))
	err := v.ValidateSlack(context.Background(), "xoxb-valid")

	s.NoError(err)
}

func (s *SlackValidatorSuite) TestSlackValidator_ConnectionRefused() {
	v := validation.NewSlackAPIValidator(validation.WithSlackBaseURL("http://127.0.0.1:1"))
	err := v.ValidateSlack(context.Background(), "some-token")

	s.Error(err)
}

func (s *SlackValidatorSuite) TestSlackValidator_NonJSONResponse() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<html><body>Not JSON</body></html>`))
	}))
	defer server.Close()

	v := validation.NewSlackAPIValidator(validation.WithSlackBaseURL(server.URL))
	err := v.ValidateSlack(context.Background(), "some-token")

	s.Error(err)
}
