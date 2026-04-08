package decisionengine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
)

// Ollama API constants
const (
	ollamaGenerateEndpoint = "/api/generate"
	jsonContentType        = "application/json"
)

// LLM prompt template for message scoring
const promptTemplate = `Score this message's importance for an ADHD user who needs to catch critical items (deadlines, outages, @mentions) without noise.

Source: %s | Sender: %s | Channel: %s
Content: %s

{"importance_score": 0-10, "confidence_score": 0.0-1.0, "reasoning": "one sentence"}`

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
	Format string `json:"format,omitempty"`
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

// createRequest builds an HTTP request for Generate without format field.
func (c *OllamaClient) createRequest(ctx context.Context, prompt string) (*http.Request, error) {
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

	return req, nil
}

// createJSONRequest builds an HTTP request for Score with format: "json".
func (c *OllamaClient) createJSONRequest(ctx context.Context, prompt string) (*http.Request, error) {
	reqBody := ollamaRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
		Format: "json",
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

	return req, nil
}

// sendRequest sends an HTTP request and returns the response body bytes.
func (c *OllamaClient) sendRequest(req *http.Request) ([]byte, error) {
	resp, err := c.httpClient.Do(req) // #nosec G704 -- URL constructed from validated config baseURL
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

	return respBytes, nil
}

// processResponse parses the outer Ollama JSON and returns the response field.
func (c *OllamaClient) processResponse(body []byte) (string, error) {
	var outerResp ollamaResponse
	if err := json.Unmarshal(body, &outerResp); err != nil {
		return "", fmt.Errorf("parsing Ollama response JSON: %w", err)
	}

	return outerResp.Response, nil
}

// processJSONResponse processes the response body and returns a ScorerResult.
func (c *OllamaClient) processJSONResponse(body []byte) (*ScorerResult, error) {
	responseText, err := c.processResponse(body)
	if err != nil {
		return nil, err
	}

	var inner scorerResponse
	if err := json.Unmarshal([]byte(responseText), &inner); err != nil {
		return nil, fmt.Errorf("parsing scorer response JSON: %w", err)
	}

	return &ScorerResult{
		ImportanceScore: inner.ImportanceScore,
		ConfidenceScore: inner.ConfidenceScore,
		Reasoning:       inner.Reasoning,
	}, nil
}

// ScoreWithContext sends a message to Ollama for scoring and returns the result.
// The examples parameter provides few-shot examples for prompt injection (not yet used).
func (c *OllamaClient) ScoreWithContext(ctx context.Context, msg *repository.Message, _ []FewShotExample) (*ScorerResult, error) {
	prompt := buildPrompt(msg)

	req, err := c.createJSONRequest(ctx, prompt)
	if err != nil {
		return nil, err
	}

	body, err := c.sendRequest(req)
	if err != nil {
		return nil, err
	}

	result, err := c.processJSONResponse(body)
	if err != nil {
		return nil, err
	}

	result.ScoringModel = c.model
	return result, nil
}

// buildPrompt constructs the LLM prompt from the message fields.
func buildPrompt(msg *repository.Message) string {
	return fmt.Sprintf(promptTemplate, msg.Source, msg.Sender, msg.Channel, msg.RawContent)
}

// Generate sends a raw prompt to Ollama and returns the response text.
func (c *OllamaClient) Generate(ctx context.Context, prompt string) (string, error) {
	req, err := c.createRequest(ctx, prompt)
	if err != nil {
		return "", err
	}

	body, err := c.sendRequest(req)
	if err != nil {
		return "", err
	}

	return c.processResponse(body)
}
