package openaichat_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"goforge.dev/cupel/components/openaichat"
)

// clearEnv wipes every environment knob openaichat reads so a test starts from a
// known baseline regardless of the developer's shell.
func clearEnv(t *testing.T) {
	t.Helper()
	t.Setenv("CUPEL_PROVIDER", "")
	t.Setenv("CUPEL_BASE_URL", "")
	t.Setenv("CUPEL_MODEL", "")
	t.Setenv("OPENAI_API_KEY", "")
}

// Example test: a full round-trip against a fake Chat Completions server,
// asserting the request path, method, headers, and JSON body shape, then
// verifying the client decodes choices[0].message.content.
func TestComplete_RoundTrip(t *testing.T) {
	clearEnv(t)
	t.Setenv("OPENAI_API_KEY", "sk-test")

	var gotAuth, gotCT string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path = %s, want /v1/chat/completions", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		gotCT = r.Header.Get("Content-Type")
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"hello from the model"}}]}`)
	}))
	defer srv.Close()

	c, err := openaichat.New("openai", "gpt-test", srv.URL+"/v1")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	out, err := c.Complete(context.Background(), "be terse", "say hi")
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if out != "hello from the model" {
		t.Errorf("Complete = %q, want %q", out, "hello from the model")
	}

	if gotAuth != "Bearer sk-test" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer sk-test")
	}
	if gotCT != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotCT)
	}
	if gotBody["model"] != "gpt-test" {
		t.Errorf("body.model = %v, want gpt-test", gotBody["model"])
	}
	msgs, ok := gotBody["messages"].([]any)
	if !ok || len(msgs) != 2 {
		t.Fatalf("body.messages = %v, want 2 messages", gotBody["messages"])
	}
	sys := msgs[0].(map[string]any)
	usr := msgs[1].(map[string]any)
	if sys["role"] != "system" || sys["content"] != "be terse" {
		t.Errorf("system message = %v", sys)
	}
	if usr["role"] != "user" || usr["content"] != "say hi" {
		t.Errorf("user message = %v", usr)
	}
}

// Example test: with no API key the Authorization header is omitted entirely, so
// a local ollama/LM Studio/vLLM server is reached unauthenticated.
func TestComplete_NoKeyOmitsAuth(t *testing.T) {
	clearEnv(t)

	var hadAuth bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, hadAuth = r.Header["Authorization"]
		io.WriteString(w, `{"choices":[{"message":{"content":"ok"}}]}`)
	}))
	defer srv.Close()

	// A custom local server: provider "ollama" with an overridden base URL and
	// no key set.
	c, err := openaichat.New("ollama", "llama3.2", srv.URL+"/v1")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := c.Complete(context.Background(), "", "hi"); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if hadAuth {
		t.Error("Authorization header was sent despite an empty key")
	}
}

// Example test: a non-200 with an error body surfaces the API message.
func TestComplete_APIError(t *testing.T) {
	clearEnv(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, `{"error":{"type":"invalid_request_error","message":"bad key"}}`)
	}))
	defer srv.Close()

	c, err := openaichat.New("ollama", "m", srv.URL+"/v1")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := c.Complete(context.Background(), "s", "u"); err == nil {
		t.Error("expected an error for a 401 response")
	}
}

// Table-driven test: provider, base-URL, and model resolution honor the
// precedence explicit argument > environment > default.
func TestNew_Resolution(t *testing.T) {
	cases := []struct {
		name                              string
		argProvider, argModel, argBase    string
		envProvider, envModel, envBase    string
		key                               string
		wantProvider, wantModel, wantBase string
		wantErr                           bool
	}{
		{
			name:         "openai defaults",
			key:          "sk-x",
			wantProvider: "openai", wantModel: "gpt-4o-mini", wantBase: "https://api.openai.com/v1",
		},
		{
			name:         "ollama defaults, no key needed",
			argProvider:  "ollama",
			wantProvider: "ollama", wantModel: "llama3.2", wantBase: "http://localhost:11434/v1",
		},
		{
			name:        "arg beats env beats default",
			argProvider: "ollama", argModel: "m-arg", argBase: "http://arg.example/v1",
			envProvider: "openai", envModel: "m-env", envBase: "http://env.example/v1",
			wantProvider: "ollama", wantModel: "m-arg", wantBase: "http://arg.example/v1",
		},
		{
			name:        "env used when arg empty",
			envProvider: "ollama", envModel: "m-env", envBase: "http://env.example/v2",
			wantProvider: "ollama", wantModel: "m-env", wantBase: "http://env.example/v2",
		},
		{
			name:        "trailing slash trimmed from base",
			argProvider: "ollama", argBase: "http://x.example/v1/",
			wantProvider: "ollama", wantModel: "llama3.2", wantBase: "http://x.example/v1",
		},
		{
			name:        "custom provider requires a base URL",
			argProvider: "vllm", argModel: "mistral",
			wantErr: true,
		},
		{
			name:        "custom provider with base URL and model resolves",
			argProvider: "vllm", argModel: "mistral", argBase: "http://localhost:8000/v1",
			wantProvider: "vllm", wantModel: "mistral", wantBase: "http://localhost:8000/v1",
		},
		{
			name:        "hosted openai without a key errors",
			argProvider: "openai",
			wantErr:     true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clearEnv(t)
			t.Setenv("CUPEL_PROVIDER", tc.envProvider)
			t.Setenv("CUPEL_MODEL", tc.envModel)
			t.Setenv("CUPEL_BASE_URL", tc.envBase)
			t.Setenv("OPENAI_API_KEY", tc.key)

			c, err := openaichat.New(tc.argProvider, tc.argModel, tc.argBase)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got provider=%q model=%q base=%q", c.Provider(), c.Model(), c.BaseURL())
				}
				return
			}
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			if c.Provider() != tc.wantProvider {
				t.Errorf("Provider = %q, want %q", c.Provider(), tc.wantProvider)
			}
			if c.Model() != tc.wantModel {
				t.Errorf("Model = %q, want %q", c.Model(), tc.wantModel)
			}
			if c.BaseURL() != tc.wantBase {
				t.Errorf("BaseURL = %q, want %q", c.BaseURL(), tc.wantBase)
			}
		})
	}
}
