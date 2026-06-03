// Package internal holds the claude component implementation: the Anthropic
// Messages API request/response handling.
package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// DefaultModel is the model used when none is configured.
const DefaultModel = "claude-opus-4-8"

const (
	endpoint   = "https://api.anthropic.com/v1/messages"
	apiVersion = "2023-06-01"
	maxTokens  = 8192
)

// Client is the concrete Anthropic client.
type Client struct {
	APIKey  string
	Model   string
	BaseURL string
	HTTP    *http.Client
}

// New reads the API key and resolves the model.
func New(model string) (Client, error) {
	key := strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY"))
	if key == "" {
		return Client{}, fmt.Errorf("claude: ANTHROPIC_API_KEY is not set")
	}
	if model == "" {
		model = os.Getenv("CUPEL_MODEL")
	}
	if model == "" {
		model = DefaultModel
	}
	return Client{
		APIKey:  key,
		Model:   model,
		BaseURL: endpoint,
		HTTP:    &http.Client{Timeout: 5 * time.Minute},
	}, nil
}

// --- request/response wire types ---

type textBlock struct {
	Type         string        `json:"type"`
	Text         string        `json:"text"`
	CacheControl *cacheControl `json:"cache_control,omitempty"`
}

type cacheControl struct {
	Type string `json:"type"`
}

type message struct {
	Role    string      `json:"role"`
	Content []textBlock `json:"content"`
}

type request struct {
	Model     string      `json:"model"`
	MaxTokens int         `json:"max_tokens"`
	System    []textBlock `json:"system"`
	Messages  []message   `json:"messages"`
}

type response struct {
	Content []textBlock `json:"content"`
	Error   *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// Complete performs one Messages API call.
func (c Client) Complete(ctx context.Context, system, user string) (string, error) {
	body := request{
		Model:     c.Model,
		MaxTokens: maxTokens,
		// Mark the (large, stable) system prompt as cacheable.
		System: []textBlock{{
			Type:         "text",
			Text:         system,
			CacheControl: &cacheControl{Type: "ephemeral"},
		}},
		Messages: []message{{
			Role:    "user",
			Content: []textBlock{{Type: "text", Text: user}},
		}},
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL, bytes.NewReader(buf))
	if err != nil {
		return "", err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", apiVersion)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("claude: request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var out response
	if err := json.Unmarshal(data, &out); err != nil {
		return "", fmt.Errorf("claude: decoding response (status %d): %w", resp.StatusCode, err)
	}
	if resp.StatusCode != http.StatusOK {
		if out.Error != nil {
			return "", fmt.Errorf("claude: API error (%s): %s", out.Error.Type, out.Error.Message)
		}
		return "", fmt.Errorf("claude: API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var sb strings.Builder
	for _, blk := range out.Content {
		if blk.Type == "text" {
			sb.WriteString(blk.Text)
		}
	}
	if sb.Len() == 0 {
		return "", fmt.Errorf("claude: empty response")
	}
	return sb.String(), nil
}
