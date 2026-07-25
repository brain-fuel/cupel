// Package internal holds the openaichat component implementation: request and
// response handling for the OpenAI Chat Completions wire format, used to reach
// OpenAI, Codex, Ollama, and any local or remote OpenAI-compatible server.
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

// Provider names understood natively. Any other value is treated as a custom
// OpenAI-compatible provider, for which a base URL must be supplied.
const (
	ProviderOpenAI = "openai"
	ProviderOllama = "ollama"
)

// Provider defaults.
const (
	openAIBaseURL   = "https://api.openai.com/v1"
	ollamaBaseURL   = "http://localhost:11434/v1"
	openAIDefModel  = "gpt-4o-mini"
	ollamaDefModel  = "llama3.2"
	completionsPath = "/chat/completions"
)

// Client is the concrete OpenAI-compatible Chat Completions client.
type Client struct {
	Provider string
	APIKey   string
	Model    string
	BaseURL  string // the API root, e.g. https://api.openai.com/v1
	HTTP     *http.Client
}

// New resolves a client from an explicit provider, model, and base URL, filling
// each empty value from the environment and then a provider default. The
// resolution precedence for every field is: explicit argument > environment >
// default.
//
//	provider  arg > CUPEL_PROVIDER > "openai"
//	base URL  arg > CUPEL_BASE_URL  > provider default (custom providers require one)
//	model     arg > CUPEL_MODEL     > provider default (custom providers require one)
//	API key   OPENAI_API_KEY (optional for ollama and for any local base URL)
func New(provider, model, baseURL string) (Client, error) {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		provider = strings.TrimSpace(os.Getenv("CUPEL_PROVIDER"))
	}
	if provider == "" {
		provider = ProviderOpenAI
	}

	base := strings.TrimSpace(baseURL)
	if base == "" {
		base = strings.TrimSpace(os.Getenv("CUPEL_BASE_URL"))
	}
	if base == "" {
		base = defaultBaseURL(provider)
	}
	if base == "" {
		return Client{}, fmt.Errorf("openaichat: no base URL for provider %q (set base-url: or CUPEL_BASE_URL)", provider)
	}
	base = strings.TrimRight(base, "/")

	if model == "" {
		model = strings.TrimSpace(os.Getenv("CUPEL_MODEL"))
	}
	if model == "" {
		model = defaultModel(provider)
	}
	if model == "" {
		return Client{}, fmt.Errorf("openaichat: no model for provider %q (set model: or CUPEL_MODEL)", provider)
	}

	key := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	// A key is only mandatory when talking to the hosted OpenAI endpoint. Local
	// servers (ollama, LM Studio, vLLM, and any custom base URL) authenticate
	// optionally or not at all, so an empty key simply omits the auth header.
	if key == "" && provider == ProviderOpenAI && base == openAIBaseURL {
		return Client{}, fmt.Errorf("openaichat: OPENAI_API_KEY is not set (required for the hosted OpenAI endpoint)")
	}

	return Client{
		Provider: provider,
		APIKey:   key,
		Model:    model,
		BaseURL:  base,
		HTTP:     &http.Client{Timeout: 5 * time.Minute},
	}, nil
}

func defaultBaseURL(provider string) string {
	switch provider {
	case ProviderOpenAI:
		return openAIBaseURL
	case ProviderOllama:
		return ollamaBaseURL
	default:
		return ""
	}
}

func defaultModel(provider string) string {
	switch provider {
	case ProviderOpenAI:
		return openAIDefModel
	case ProviderOllama:
		return ollamaDefModel
	default:
		return ""
	}
}

// --- request/response wire types ---

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// Complete performs one Chat Completions call, sending the system and user
// prompts as the first two messages and returning choices[0].message.content.
func (c Client) Complete(ctx context.Context, system, user string) (string, error) {
	msgs := make([]chatMessage, 0, 2)
	if system != "" {
		msgs = append(msgs, chatMessage{Role: "system", Content: system})
	}
	msgs = append(msgs, chatMessage{Role: "user", Content: user})

	buf, err := json.Marshal(chatRequest{Model: c.Model, Messages: msgs})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+completionsPath, bytes.NewReader(buf))
	if err != nil {
		return "", err
	}
	req.Header.Set("content-type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("openaichat: request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var out chatResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return "", fmt.Errorf("openaichat: decoding response (status %d): %w", resp.StatusCode, err)
	}
	if resp.StatusCode != http.StatusOK {
		if out.Error != nil {
			return "", fmt.Errorf("openaichat: API error (%s): %s", out.Error.Type, out.Error.Message)
		}
		return "", fmt.Errorf("openaichat: API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	if len(out.Choices) == 0 {
		return "", fmt.Errorf("openaichat: empty response (no choices)")
	}
	content := out.Choices[0].Message.Content
	if strings.TrimSpace(content) == "" {
		return "", fmt.Errorf("openaichat: empty response")
	}
	return content, nil
}
