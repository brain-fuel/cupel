package prompt_test

import (
	"strings"
	"testing"

	"goforge.dev/cupel/components/prompt"
	"goforge.dev/cupel/components/spec"
)

func mustSpec(t *testing.T) spec.Spec {
	t.Helper()
	s, err := spec.Parse("user", "", "A user has a non-empty name and an email containing '@'.")
	if err != nil {
		t.Fatalf("spec.Parse: %v", err)
	}
	return s
}

// Example test.
func TestUser_MentionsTestPackageAndSpec(t *testing.T) {
	u := prompt.User(mustSpec(t))
	if !strings.Contains(u, "package user_test") {
		t.Errorf("user prompt missing required package directive:\n%s", u)
	}
	if !strings.Contains(u, "containing '@'") {
		t.Error("user prompt did not embed the specification text")
	}
}

// Table-driven test: the system prompt names all three test categories.
func TestSystem_CoversAllThreeKinds(t *testing.T) {
	sys := prompt.System()
	for _, want := range []string{"EXAMPLE TESTS", "DATA-DRIVEN", "PROPERTY-BASED", "testing/quick"} {
		if !strings.Contains(sys, want) {
			t.Errorf("system prompt missing %q", want)
		}
	}
}
