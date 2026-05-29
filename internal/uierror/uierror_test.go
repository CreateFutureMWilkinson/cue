package uierror_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/uierror"
	"github.com/CreateFutureMWilkinson/cue/pkg/client"
)

type ClassifySuite struct {
	suite.Suite
}

func TestClassify(t *testing.T) {
	suite.Run(t, new(ClassifySuite))
}

// TestClassifyByCode is the table-driven sweep across every API error
// code. Each row pins the resulting Kind and the salient flag (e.g.,
// ActionRetryRePair on UNAUTHORIZED).
func (s *ClassifySuite) TestClassifyByCode() {
	cases := []struct {
		code            string
		message         string
		wantKind        uierror.Kind
		wantRetryRePair bool
		wantBodyHas     string
	}{
		{
			code:            client.ErrCodeUnauthorized,
			message:         "stale token",
			wantKind:        uierror.KindUnauthorized,
			wantRetryRePair: true,
			wantBodyHas:     "Restart Cue and re-pair",
		},
		{
			code:        client.ErrCodeNotFound,
			message:     "no such message",
			wantKind:    uierror.KindNotFound,
			wantBodyHas: "no such message",
		},
		{
			code:        client.ErrCodeNotFound,
			message:     "", // server omitted message
			wantKind:    uierror.KindNotFound,
			wantBodyHas: "no longer available",
		},
		{
			code:        client.ErrCodeValidation,
			message:     "name required",
			wantKind:    uierror.KindValidation,
			wantBodyHas: "name required",
		},
		{
			code:        client.ErrCodeConflict,
			message:     "version mismatch",
			wantKind:    uierror.KindConflict,
			wantBodyHas: "version mismatch",
		},
		{
			code:        client.ErrCodeServerError,
			message:     "stack trace omitted",
			wantKind:    uierror.KindServerError,
			wantBodyHas: "Try again",
		},
		{
			code:        client.ErrCodeClientError,
			message:     "missing header",
			wantKind:    uierror.KindClientError,
			wantBodyHas: "missing header",
		},
		{
			code:        "TOTALLY_NOVEL_CODE",
			message:     "future error",
			wantKind:    uierror.KindUnknown,
			wantBodyHas: "future error",
		},
	}

	for _, tc := range cases {
		tc := tc
		s.Run(tc.code+"_"+tc.message, func() {
			apiErr := &client.APIError{Code: tc.code, Message: tc.message}
			got := uierror.Classify(apiErr)
			s.Equal(tc.wantKind, got.Kind, "Kind mismatch")
			s.Equal(tc.wantRetryRePair, got.ActionRetryRePair, "ActionRetryRePair mismatch")
			s.NotEmpty(got.Title, "every classification must populate Title")
			if tc.wantBodyHas != "" {
				s.Contains(got.Body, tc.wantBodyHas)
			}
			s.Equal(apiErr, got.Underlying, "Underlying must equal the input error")
		})
	}
}

// TestClassifyWrappedError ensures Classify peels the error chain via
// errors.As — adapters wrap with fmt.Errorf("...: %w", err) before
// returning, so the helper must look through the wrap.
func (s *ClassifySuite) TestClassifyWrappedError() {
	apiErr := &client.APIError{Code: client.ErrCodeUnauthorized, Message: "expired"}
	wrapped := fmt.Errorf("resolve notification %s: %w", "<id>", apiErr)

	got := uierror.Classify(wrapped)
	s.Equal(uierror.KindUnauthorized, got.Kind)
	s.True(got.ActionRetryRePair)
	s.Equal(wrapped, got.Underlying, "Underlying must be the outermost wrap so logs see full context")
}

// TestClassifyNonAPIError falls through to KindUnknown without a
// panic on transport / programmer errors.
func (s *ClassifySuite) TestClassifyNonAPIError() {
	got := uierror.Classify(errors.New("connection refused"))
	s.Equal(uierror.KindUnknown, got.Kind)
	s.Contains(got.Body, "connection refused")
	s.False(got.ActionRetryRePair)
}

// TestClassifyNilReturnsZero is a defensive check: callers should not
// pass nil, but if they do they must get a Display with no surprise
// strings to render.
func (s *ClassifySuite) TestClassifyNilReturnsZero() {
	got := uierror.Classify(nil)
	s.Equal(uierror.KindUnknown, got.Kind)
	s.Empty(got.Title)
	s.Empty(got.Body)
	s.False(got.ActionRetryRePair)
	s.Nil(got.Underlying)
}
