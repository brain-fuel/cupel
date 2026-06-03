package architecture_test

import (
	"strings"
	"testing"

	"goforge.dev/cupel/components/architecture"
	"goforge.dev/cupel/components/spec"
)

func userSpec(t *testing.T) spec.Spec {
	t.Helper()
	s, err := spec.Parse("user", "", "A user has a non-empty name.")
	if err != nil {
		t.Fatalf("spec.Parse: %v", err)
	}
	return s
}

// Example test: a fenced, unformatted but valid reply is accepted and gofmt'd.
func TestEnforceTests_StripsFenceAndFormats(t *testing.T) {
	raw := "```go\npackage user_test\nimport \"testing\"\nfunc TestX(t *testing.T){_=t}\n```"
	got, err := architecture.EnforceTests(userSpec(t), raw)
	if err != nil {
		t.Fatalf("EnforceTests: %v", err)
	}
	if strings.Contains(got, "```") {
		t.Error("fence not stripped")
	}
	if !strings.Contains(got, "func TestX(t *testing.T) {") {
		t.Errorf("output not gofmt-formatted:\n%s", got)
	}
}

// Table-driven test: rejection rules.
func TestEnforceTests_Rejects(t *testing.T) {
	cases := []struct {
		name string
		raw  string
	}{
		{"empty", ""},
		{"wrong package", "package user\nfunc f(){}"},
		{"not go", "this is not go at all {{{"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := architecture.EnforceTests(userSpec(t), tc.raw); err == nil {
				t.Errorf("expected error for %q", tc.name)
			}
		})
	}
}

// Example test: the rendered brick is a complete, check-valid component.
func TestBrick_RendersValidComponentLayout(t *testing.T) {
	tests, err := architecture.EnforceTests(userSpec(t), "package user_test")
	if err != nil {
		t.Fatalf("EnforceTests: %v", err)
	}
	files := architecture.Brick(userSpec(t), "goforge.dev/demo", tests)

	want := map[string]bool{
		"user_test.go":     true,
		"user.go":          false,
		"internal/core.go": false,
		"component.yaml":   false,
	}
	got := map[string]bool{}
	for _, f := range files {
		got[f.RelPath] = f.Generated
		if strings.TrimSpace(f.Content) == "" {
			t.Errorf("file %q is empty", f.RelPath)
		}
	}
	for path, gen := range want {
		g, ok := got[path]
		if !ok {
			t.Errorf("missing rendered file %q", path)
			continue
		}
		if g != gen {
			t.Errorf("file %q Generated = %v, want %v", path, g, gen)
		}
	}
}
