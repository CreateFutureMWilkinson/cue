package validation_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CreateFutureMWilkinson/cue/internal/service/validation"
	"github.com/stretchr/testify/suite"
)

type ICSValidatorSuite struct {
	suite.Suite
}

func TestICSValidator(t *testing.T) {
	suite.Run(t, new(ICSValidatorSuite))
}

func (s *ICSValidatorSuite) TestICSValidator_ValidCalendar() {
	icsContent := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//Test//Test//EN\r\nEND:VCALENDAR"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(icsContent))
	}))
	defer server.Close()

	v := validation.NewICSValidator(validation.WithHTTPClient(server.Client()))
	err := v.ValidateCalendar(context.Background(), server.URL)

	s.NoError(err)
}

func (s *ICSValidatorSuite) TestICSValidator_NotICalendar() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body>Not a calendar</body></html>"))
	}))
	defer server.Close()

	v := validation.NewICSValidator(validation.WithHTTPClient(server.Client()))
	err := v.ValidateCalendar(context.Background(), server.URL)

	s.Error(err)
	s.Contains(err.Error(), "invalid iCalendar")
}

func (s *ICSValidatorSuite) TestICSValidator_HTTP404() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	v := validation.NewICSValidator(validation.WithHTTPClient(server.Client()))
	err := v.ValidateCalendar(context.Background(), server.URL)

	s.Error(err)
	s.Contains(err.Error(), "404")
}

func (s *ICSValidatorSuite) TestICSValidator_ConnectionRefused() {
	v := validation.NewICSValidator()
	err := v.ValidateCalendar(context.Background(), "http://127.0.0.1:1")

	s.Error(err)
}
