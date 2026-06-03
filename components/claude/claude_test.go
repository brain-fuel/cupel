package claude_test

import (
	"testing"

	"goforge.dev/cupel/components/claude"
)

// Example test: an explicit model wins over the default.
func TestNew_UsesProvidedModel(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	t.Setenv("CUPEL_MODEL", "")
	c, err := claude.New("claude-sonnet-4-6")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if c.Model() != "claude-sonnet-4-6" {
		t.Errorf("Model = %q, want %q", c.Model(), "claude-sonnet-4-6")
	}
}

// Table-driven test: model resolution precedence (arg > CUPEL_MODEL > default).
func TestNew_ModelPrecedence(t *testing.T) {
	cases := []struct {
		name     string
		arg      string
		envModel string
		want     string
	}{
		{"arg wins", "m-arg", "m-env", "m-arg"},
		{"env fallback", "", "m-env", "m-env"},
		{"default", "", "", claude.DefaultModel},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("ANTHROPIC_API_KEY", "test-key")
			t.Setenv("CUPEL_MODEL", tc.envModel)
			c, err := claude.New(tc.arg)
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			if c.Model() != tc.want {
				t.Errorf("Model = %q, want %q", c.Model(), tc.want)
			}
		})
	}
}

// Example test: missing API key is a hard error.
func TestNew_RequiresAPIKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	if _, err := claude.New("m"); err == nil {
		t.Error("expected error when ANTHROPIC_API_KEY is unset")
	}
}
