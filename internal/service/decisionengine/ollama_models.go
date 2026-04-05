package decisionengine

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

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

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/tags", nil)
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
		lookup := model
		if !strings.Contains(model, ":") {
			lookup = model + ":latest"
		}
		if _, ok := available[lookup]; !ok {
			missing = append(missing, model)
		}
	}

	if len(missing) == 0 {
		return nil
	}

	pullCmds := make([]string, len(missing))
	for i, m := range missing {
		pullCmds[i] = "ollama pull " + m
	}

	if len(missing) == 1 {
		return fmt.Errorf("missing Ollama model %q; run: %s", missing[0], pullCmds[0])
	}

	quoted := make([]string, len(missing))
	for i, m := range missing {
		quoted[i] = fmt.Sprintf("%q", m)
	}
	return fmt.Errorf("missing Ollama models: %s; run: %s", strings.Join(quoted, ", "), strings.Join(pullCmds, " && "))
}
