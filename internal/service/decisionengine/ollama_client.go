package decisionengine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
)

// Ollama API constants
const (
	ollamaGenerateEndpoint = "/api/generate"
	jsonContentType        = "application/json"
	markdownFence          = "```"
)

// LLM prompt template for message scoring
const promptTemplate = `You are an ADHD-friendly message importance scorer. Evaluate the following message and return a JSON object with these fields:
- importance_score: a float from 0 to 10 indicating how important/urgent this message is
- confidence_score: a float from 0.0 to 1.0 indicating your confidence in the rating
- reasoning: a brief explanation of your rating

Message details:
- Source: %s
- Sender: %s
- Channel: %s
- Content: %s

Respond ONLY with valid JSON. Do not include any other text.`

// OllamaClient communicates with a local Ollama instance to score messages.
type OllamaClient struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewOllamaClient creates a new OllamaClient with the given base URL, model name, and timeout.
// Returns an error if any parameter is invalid.
func NewOllamaClient(baseURL string, model string, timeout time.Duration) (*OllamaClient, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("baseURL must not be empty")
	}
	if model == "" {
		return nil, fmt.Errorf("model must not be empty")
	}
	if timeout <= 0 {
		return nil, fmt.Errorf("timeout must be greater than zero")
	}

	return &OllamaClient{
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// ollamaRequest is the JSON body sent to the Ollama /api/generate endpoint.
type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// ollamaResponse is the outer JSON response from Ollama.
type ollamaResponse struct {
	Response string `json:"response"`
}

// scorerResponse is the inner JSON parsed from the Ollama response field.
type scorerResponse struct {
	ImportanceScore float64 `json:"importance_score"`
	ConfidenceScore float64 `json:"confidence_score"`
	Reasoning       string  `json:"reasoning"`
}

// Score sends a message to Ollama for scoring and returns the result.
func (c *OllamaClient) Score(ctx context.Context, msg *repository.Message) (*ScorerResult, error) {
	prompt := buildPrompt(msg)

	reqBody := ollamaRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshalling Ollama request: %w", err)
	}

	url := c.baseURL + ollamaGenerateEndpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("creating HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", jsonContentType)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending HTTP request to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama API returned non-200 status: %d", resp.StatusCode)
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading Ollama response body: %w", err)
	}

	var outerResp ollamaResponse
	if err := json.Unmarshal(respBytes, &outerResp); err != nil {
		return nil, fmt.Errorf("parsing Ollama response JSON: %w", err)
	}

	innerJSON := extractJSON(outerResp.Response)

	var inner scorerResponse
	if err := json.Unmarshal([]byte(innerJSON), &inner); err != nil {
		return nil, fmt.Errorf("parsing scorer response JSON: %w", err)
	}

	return &ScorerResult{
		ImportanceScore: inner.ImportanceScore,
		ConfidenceScore: inner.ConfidenceScore,
		Reasoning:       inner.Reasoning,
	}, nil
}

// buildPrompt constructs the LLM prompt from the message fields.
func buildPrompt(msg *repository.Message) string {
	return fmt.Sprintf(promptTemplate, msg.Source, msg.Sender, msg.Channel, msg.RawContent)
}

// extractJSON strips markdown code block wrapping if present.
// Some LLMs wrap JSON responses in markdown code blocks like ```json...```
func extractJSON(s string) string {
	trimmed := strings.TrimSpace(s)
	if strings.HasPrefix(trimmed, markdownFence) {
		// Remove opening fence (with optional language tag)
		idx := strings.Index(trimmed, "\n")
		if idx >= 0 {
			trimmed = trimmed[idx+1:]
		}
		// Remove closing fence
		if lastIdx := strings.LastIndex(trimmed, markdownFence); lastIdx >= 0 {
			trimmed = trimmed[:lastIdx]
		}
		trimmed = strings.TrimSpace(trimmed)
	}
	return trimmed
}
