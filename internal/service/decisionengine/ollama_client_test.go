package decisionengine_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/service/decisionengine"

	"github.com/stretchr/testify/suite"
)

// --- Suite ---

type OllamaClientSuite struct {
	suite.Suite
}

func TestOllamaClient(t *testing.T) {
	suite.Run(t, new(OllamaClientSuite))
}

// --- Constructor validation ---

func (s *OllamaClientSuite) TestNewOllamaClient_ValidInputs() {
	client, err := decisionengine.NewOllamaClient("http://localhost:11434", "neural-chat", 10*time.Second)
	s.NoError(err)
	s.NotNil(client)
}

func (s *OllamaClientSuite) TestNewOllamaClient_EmptyBaseURL() {
	_, err := decisionengine.NewOllamaClient("", "neural-chat", 10*time.Second)
	s.Error(err)
	s.Contains(err.Error(), "baseURL")
}

func (s *OllamaClientSuite) TestNewOllamaClient_EmptyModel() {
	_, err := decisionengine.NewOllamaClient("http://localhost:11434", "", 10*time.Second)
	s.Error(err)
	s.Contains(err.Error(), "model")
}

func (s *OllamaClientSuite) TestNewOllamaClient_ZeroTimeout() {
	_, err := decisionengine.NewOllamaClient("http://localhost:11434", "neural-chat", 0)
	s.Error(err)
	s.Contains(err.Error(), "timeout")
}

// --- Implements Scorer interface ---

func (s *OllamaClientSuite) TestOllamaClient_ImplementsScorer() {
	client, err := decisionengine.NewOllamaClient("http://localhost:11434", "neural-chat", 10*time.Second)
	s.Require().NoError(err)

	// Verify it satisfies the Scorer interface at compile time
	var _ decisionengine.Scorer = client
}

// --- Successful scoring ---

func (s *OllamaClientSuite) TestScore_ValidResponse_ReturnsScores() {
	ollamaResponse := map[string]any{
		"response": `{"importance_score": 8.5, "confidence_score": 0.9, "reasoning": "server outage detected"}`,
		"done":     true,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ollamaResponse)
	}))
	defer server.Close()

	client, err := decisionengine.NewOllamaClient(server.URL, "neural-chat", 10*time.Second)
	s.Require().NoError(err)

	msg := &repository.Message{
		Source:     "slack",
		Sender:     "bob",
		Channel:    "ops-alerts",
		RawContent: "production database is down, all services affected",
	}

	result, err := client.Score(context.Background(), msg)
	s.NoError(err)
	s.NotNil(result)
	s.Equal(8.5, result.ImportanceScore)
	s.Equal(0.9, result.ConfidenceScore)
	s.Equal("server outage detected", result.Reasoning)
}

// --- Request format ---

func (s *OllamaClientSuite) TestScore_SendsCorrectRequestToOllama() {
	var receivedBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify HTTP method and path
		s.Equal("POST", r.Method)
		s.Equal("/api/generate", r.URL.Path)

		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)

		resp := map[string]any{
			"response": `{"importance_score": 5.0, "confidence_score": 0.7, "reasoning": "normal message"}`,
			"done":     true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := decisionengine.NewOllamaClient(server.URL, "neural-chat", 10*time.Second)
	s.Require().NoError(err)

	msg := &repository.Message{
		Source:     "slack",
		Sender:     "alice",
		Channel:    "general",
		RawContent: "has anyone seen the new design?",
	}

	_, err = client.Score(context.Background(), msg)
	s.NoError(err)

	// Verify model is set correctly
	s.Equal("neural-chat", receivedBody["model"])

	// Verify prompt contains message context
	prompt, ok := receivedBody["prompt"].(string)
	s.True(ok, "prompt should be a string")
	s.Contains(prompt, "alice", "prompt should contain sender")
	s.Contains(prompt, "general", "prompt should contain channel")
	s.Contains(prompt, "has anyone seen the new design?", "prompt should contain message content")

	// Verify stream is disabled (we want complete response)
	s.Equal(false, receivedBody["stream"])
}

// --- Prompt includes source context ---

func (s *OllamaClientSuite) TestScore_PromptIncludesSource() {
	var receivedBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)

		resp := map[string]any{
			"response": `{"importance_score": 5.0, "confidence_score": 0.7, "reasoning": "normal"}`,
			"done":     true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := decisionengine.NewOllamaClient(server.URL, "neural-chat", 10*time.Second)
	s.Require().NoError(err)

	msg := &repository.Message{
		Source:     "email",
		Sender:     "boss@company.com",
		Channel:    "inbox",
		RawContent: "urgent: need response by EOD",
	}

	_, err = client.Score(context.Background(), msg)
	s.NoError(err)

	prompt := receivedBody["prompt"].(string)
	s.Contains(prompt, "email", "prompt should contain source type")
}

// --- Timeout handling ---

func (s *OllamaClientSuite) TestScore_ContextTimeout_ReturnsError() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(500 * time.Millisecond)
		resp := map[string]any{
			"response": `{"importance_score": 5.0, "confidence_score": 0.7, "reasoning": "normal"}`,
			"done":     true,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := decisionengine.NewOllamaClient(server.URL, "neural-chat", 10*time.Second)
	s.Require().NoError(err)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	msg := &repository.Message{
		Source:     "slack",
		RawContent: "some message",
	}

	_, err = client.Score(ctx, msg)
	s.Error(err, "should return error on context timeout")
}

// --- Invalid JSON in response field ---

func (s *OllamaClientSuite) TestScore_InvalidJSONInResponse_ReturnsError() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"response": "this is not valid json at all",
			"done":     true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := decisionengine.NewOllamaClient(server.URL, "neural-chat", 10*time.Second)
	s.Require().NoError(err)

	msg := &repository.Message{
		Source:     "slack",
		RawContent: "test message",
	}

	_, err = client.Score(context.Background(), msg)
	s.Error(err, "should return error when Ollama response is not valid JSON")
}

// --- Non-200 status code ---

func (s *OllamaClientSuite) TestScore_Non200StatusCode_ReturnsError() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "internal server error")
	}))
	defer server.Close()

	client, err := decisionengine.NewOllamaClient(server.URL, "neural-chat", 10*time.Second)
	s.Require().NoError(err)

	msg := &repository.Message{
		Source:     "slack",
		RawContent: "test message",
	}

	_, err = client.Score(context.Background(), msg)
	s.Error(err, "should return error on non-200 status code")
	s.Contains(err.Error(), "500")
}

// --- Invalid outer JSON from Ollama ---

func (s *OllamaClientSuite) TestScore_InvalidOuterJSON_ReturnsError() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "not json at all {{{")
	}))
	defer server.Close()

	client, err := decisionengine.NewOllamaClient(server.URL, "neural-chat", 10*time.Second)
	s.Require().NoError(err)

	msg := &repository.Message{
		Source:     "slack",
		RawContent: "test message",
	}

	_, err = client.Score(context.Background(), msg)
	s.Error(err, "should return error when Ollama returns invalid JSON")
}

// --- Connection refused ---

func (s *OllamaClientSuite) TestScore_ConnectionRefused_ReturnsError() {
	client, err := decisionengine.NewOllamaClient("http://localhost:1", "neural-chat", 2*time.Second)
	s.Require().NoError(err)

	msg := &repository.Message{
		Source:     "slack",
		RawContent: "test message",
	}

	_, err = client.Score(context.Background(), msg)
	s.Error(err, "should return error when cannot connect to Ollama")
}

// --- Response with extra text around JSON ---

func (s *OllamaClientSuite) TestScore_ResponseWithMarkdownWrapping_ExtractsJSON() {
	// Some LLMs wrap JSON in markdown code blocks
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"response": "```json\n{\"importance_score\": 6.0, \"confidence_score\": 0.85, \"reasoning\": \"meeting reminder\"}\n```",
			"done":     true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := decisionengine.NewOllamaClient(server.URL, "neural-chat", 10*time.Second)
	s.Require().NoError(err)

	msg := &repository.Message{
		Source:     "email",
		RawContent: "reminder: team standup at 10am",
	}

	result, err := client.Score(context.Background(), msg)
	s.NoError(err)
	s.Equal(6.0, result.ImportanceScore)
	s.Equal(0.85, result.ConfidenceScore)
	s.Equal("meeting reminder", result.Reasoning)
}

// --- Prompt requests JSON format ---

func (s *OllamaClientSuite) TestScore_PromptRequestsJSONFormat() {
	var receivedBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)

		resp := map[string]any{
			"response": `{"importance_score": 5.0, "confidence_score": 0.7, "reasoning": "normal"}`,
			"done":     true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := decisionengine.NewOllamaClient(server.URL, "neural-chat", 10*time.Second)
	s.Require().NoError(err)

	msg := &repository.Message{Source: "slack", RawContent: "hello"}

	_, err = client.Score(context.Background(), msg)
	s.NoError(err)

	prompt := receivedBody["prompt"].(string)
	// Prompt should instruct LLM to return JSON with the expected fields
	s.True(
		strings.Contains(prompt, "importance_score") && strings.Contains(prompt, "confidence_score") && strings.Contains(prompt, "reasoning"),
		"prompt should request JSON with importance_score, confidence_score, and reasoning fields",
	)
}
