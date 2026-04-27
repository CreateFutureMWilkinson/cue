package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

// APILintSuite covers the OpenAPI validator used by `just api-lint`.
type APILintSuite struct {
	suite.Suite
}

func TestAPILint(t *testing.T) {
	suite.Run(t, new(APILintSuite))
}

// TestValidSpecReturnsNoError asserts that Validate returns nil for a
// minimal, valid OpenAPI 3.1 document. Against the RED-phase stub
// (which returns ErrNotImplemented) this test is expected to FAIL.
func (s *APILintSuite) TestValidSpecReturnsNoError() {
	spec := `openapi: 3.1.0
info:
  title: Cue API
  version: 0.1.0
paths: {}
`
	dir := s.T().TempDir()
	path := filepath.Join(dir, "openapi.yaml")
	s.Require().NoError(os.WriteFile(path, []byte(spec), 0o600))

	err := Validate(context.Background(), path)
	s.NoError(err)
}
