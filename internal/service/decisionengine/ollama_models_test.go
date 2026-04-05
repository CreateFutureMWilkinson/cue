package decisionengine_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CreateFutureMWilkinson/cue/internal/service/decisionengine"

	"github.com/stretchr/testify/suite"
)

// --- Suite ---

type OllamaModelsSuite struct {
	suite.Suite
}

func TestOllamaModels(t *testing.T) {
	suite.Run(t, new(OllamaModelsSuite))
}

// helper: create a test server that returns an OllamaTagsResponse with the given model names.
func (s *OllamaModelsSuite) ollamaTagsServer(models []string) *httptest.Server {
	type ollamaModel struct {
		Name string `json:"name"`
	}
	type ollamaTagsResponse struct {
		Models []ollamaModel `json:"models"`
	}

	resp := ollamaTagsResponse{}
	for _, m := range models {
		resp.Models = append(resp.Models, ollamaModel{Name: m})
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("/api/tags", r.URL.Path, "expected request to /api/tags")
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(resp)
		s.Require().NoError(err)
	}))
}

// --- Test: Both models present — no error ---

func (s *OllamaModelsSuite) TestBothModelsPresent() {
	srv := s.ollamaTagsServer([]string{"neural-chat:latest", "nomic-embed-text:latest"})
	defer srv.Close()

	err := decisionengine.ValidateOllamaModels(
		context.Background(),
		srv.URL,
		[]string{"neural-chat:latest", "nomic-embed-text:latest"},
	)
	s.Require().NoError(err)
}

// --- Test: One model missing — error names the missing model ---

func (s *OllamaModelsSuite) TestOneModelMissing() {
	srv := s.ollamaTagsServer([]string{"neural-chat:latest"})
	defer srv.Close()

	err := decisionengine.ValidateOllamaModels(
		context.Background(),
		srv.URL,
		[]string{"neural-chat:latest", "nomic-embed-text:latest"},
	)
	s.Require().Error(err)
	s.Contains(err.Error(), "nomic-embed-text:latest")
	s.NotContains(err.Error(), "neural-chat")
	s.Contains(err.Error(), "ollama pull")
}

// --- Test: Both models missing — error names both ---

func (s *OllamaModelsSuite) TestBothModelsMissing() {
	srv := s.ollamaTagsServer([]string{"some-other-model:latest"})
	defer srv.Close()

	err := decisionengine.ValidateOllamaModels(
		context.Background(),
		srv.URL,
		[]string{"neural-chat:latest", "nomic-embed-text:latest"},
	)
	s.Require().Error(err)
	s.Contains(err.Error(), "neural-chat:latest")
	s.Contains(err.Error(), "nomic-embed-text:latest")
	s.Contains(err.Error(), "ollama pull")
}

// --- Test: Tag matching — config "neural-chat" matches API "neural-chat:latest" ---

func (s *OllamaModelsSuite) TestImplicitLatestTag() {
	srv := s.ollamaTagsServer([]string{"neural-chat:latest", "nomic-embed-text:latest"})
	defer srv.Close()

	err := decisionengine.ValidateOllamaModels(
		context.Background(),
		srv.URL,
		[]string{"neural-chat", "nomic-embed-text"},
	)
	s.Require().NoError(err)
}

// --- Test: Tag matching — config "neural-chat:v2" matches exactly ---

func (s *OllamaModelsSuite) TestExactTagMatch() {
	srv := s.ollamaTagsServer([]string{"neural-chat:v2", "nomic-embed-text:latest"})
	defer srv.Close()

	err := decisionengine.ValidateOllamaModels(
		context.Background(),
		srv.URL,
		[]string{"neural-chat:v2", "nomic-embed-text"},
	)
	s.Require().NoError(err)
}

// --- Test: Exact tag mismatch — config "neural-chat:v2" does NOT match "neural-chat:latest" ---

func (s *OllamaModelsSuite) TestExactTagMismatch() {
	srv := s.ollamaTagsServer([]string{"neural-chat:latest", "nomic-embed-text:latest"})
	defer srv.Close()

	err := decisionengine.ValidateOllamaModels(
		context.Background(),
		srv.URL,
		[]string{"neural-chat:v2", "nomic-embed-text"},
	)
	s.Require().Error(err)
	s.Contains(err.Error(), "neural-chat:v2")
}

// --- Test: Ollama unreachable — no error (warning only) ---

func (s *OllamaModelsSuite) TestOllamaUnreachable() {
	// Use a URL that will refuse connections.
	err := decisionengine.ValidateOllamaModels(
		context.Background(),
		"http://127.0.0.1:0",
		[]string{"neural-chat", "nomic-embed-text"},
	)
	s.Require().NoError(err, "unreachable Ollama should return nil (warning only)")
}

// --- Test: Ollama returns invalid JSON — no error (warning only) ---

func (s *OllamaModelsSuite) TestInvalidJSON() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not valid json{{{"))
	}))
	defer srv.Close()

	err := decisionengine.ValidateOllamaModels(
		context.Background(),
		srv.URL,
		[]string{"neural-chat"},
	)
	s.Require().NoError(err, "invalid JSON should return nil (warning only)")
}

// --- Test: Ollama returns empty model list — error for all requested models ---

func (s *OllamaModelsSuite) TestEmptyModelList() {
	srv := s.ollamaTagsServer([]string{})
	defer srv.Close()

	err := decisionengine.ValidateOllamaModels(
		context.Background(),
		srv.URL,
		[]string{"neural-chat", "nomic-embed-text"},
	)
	s.Require().Error(err)
	s.Contains(err.Error(), "neural-chat")
	s.Contains(err.Error(), "nomic-embed-text")
	s.Contains(err.Error(), "ollama pull")
}

// --- Test: Context cancellation — returns context error ---

func (s *OllamaModelsSuite) TestContextCancellation() {
	srv := s.ollamaTagsServer([]string{"neural-chat:latest"})
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := decisionengine.ValidateOllamaModels(
		ctx,
		srv.URL,
		[]string{"neural-chat"},
	)
	s.Require().Error(err)
	s.ErrorIs(err, context.Canceled)
}

// --- Test: Empty model list input — no error (nothing to validate) ---

func (s *OllamaModelsSuite) TestEmptyModelListInput() {
	// Should not even contact Ollama — no server needed.
	err := decisionengine.ValidateOllamaModels(
		context.Background(),
		"http://127.0.0.1:0",
		[]string{},
	)
	s.Require().NoError(err)
}

// --- Test: Nil model list input — no error (nothing to validate) ---

func (s *OllamaModelsSuite) TestNilModelListInput() {
	err := decisionengine.ValidateOllamaModels(
		context.Background(),
		"http://127.0.0.1:0",
		nil,
	)
	s.Require().NoError(err)
}
