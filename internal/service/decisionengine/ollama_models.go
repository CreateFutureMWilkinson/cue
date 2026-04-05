package decisionengine

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const ollamaTagsEndpoint = "/api/tags"

type ollamaModel struct {
	Name string `json:"name"`
}

type ollamaTagsResponse struct {
	Models []ollamaModel `json:"models"`
}

// ValidateOllamaModels checks that the requested models are available in the
// Ollama instance at baseURL. If Ollama is unreachable or returns invalid JSON,
// nil is returned (transient/unexpected condition). If models are missing, an
// error listing them with pull instructions is returned.
func ValidateOllamaModels(ctx context.Context, baseURL string, models []string) error {
	if len(models) == 0 {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+ollamaTagsEndpoint, nil)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("validate ollama models: %w", ctx.Err())
		}
		return nil
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("validate ollama models: %w", ctx.Err())
		}
		// Unreachable — transient, return nil.
		return nil
	}
	defer resp.Body.Close()

	var tags ollamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		// Invalid JSON — unexpected, return nil.
		return nil
	}

	available := make(map[string]struct{}, len(tags.Models))
	for _, m := range tags.Models {
		available[m.Name] = struct{}{}
	}

	var missing []string
	for _, model := range models {
		if !isModelAvailable(model, available) {
			missing = append(missing, model)
		}
	}

	if len(missing) == 0 {
		return nil
	}

	return formatMissingModelsError(missing)
}

// isModelAvailable checks if a model is available in the map of available models,
// adding ":latest" tag if no tag is specified in the model name.
func isModelAvailable(model string, available map[string]struct{}) bool {
	lookupName := model
	if !strings.Contains(model, ":") {
		lookupName = model + ":latest"
	}
	_, exists := available[lookupName]
	return exists
}

// formatMissingModelsError creates a user-friendly error message for missing models
// with appropriate pull commands.
func formatMissingModelsError(missing []string) error {
	pullCmds := make([]string, len(missing))
	for i, model := range missing {
		pullCmds[i] = "ollama pull " + model
	}

	if len(missing) == 1 {
		return fmt.Errorf("missing Ollama model %q; run: %s", missing[0], pullCmds[0])
	}

	quotedModels := make([]string, len(missing))
	for i, model := range missing {
		quotedModels[i] = fmt.Sprintf("%q", model)
	}

	return fmt.Errorf("missing Ollama models: %s; run: %s",
		strings.Join(quotedModels, ", "),
		strings.Join(pullCmds, " && "))
}
