package claudecode_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"goforge.dev/cupel/components/claudecode"
)

// fakeBin writes an executable stub that echoes script to stdout and returns its
// path. The test points CUPEL_CLAUDE_BIN at it so no real CLI is needed.
func fakeBin(t *testing.T, script string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake-binary shell stub is POSIX-only")
	}
	path := filepath.Join(t.TempDir(), "claude")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// Example test: New fails when the binary is absent.
func TestNew_MissingBinary(t *testing.T) {
	t.Setenv("CUPEL_CLAUDE_BIN", filepath.Join(t.TempDir(), "does-not-exist"))
	if _, err := claudecode.New(""); err == nil {
		t.Error("expected error for missing claude binary")
	}
}

// Table-driven test: model label resolution.
func TestNew_ModelLabel(t *testing.T) {
	bin := fakeBin(t, "true")
	cases := []struct {
		name     string
		arg      string
		envModel string
		want     string
	}{
		{"arg wins", "m-arg", "m-env", "m-arg"},
		{"env fallback", "", "m-env", "m-env"},
		{"cli default", "", "", "(claude-code default)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("CUPEL_CLAUDE_BIN", bin)
			t.Setenv("CUPEL_MODEL", tc.envModel)
			c, err := claudecode.New(tc.arg)
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			if c.Model() != tc.want {
				t.Errorf("Model = %q, want %q", c.Model(), tc.want)
			}
		})
	}
}

// Example test: Complete returns the CLI's stdout.
func TestComplete_ReturnsStdout(t *testing.T) {
	t.Setenv("CUPEL_CLAUDE_BIN", fakeBin(t, `echo "package user_test"`))
	t.Setenv("CUPEL_MODEL", "")
	c, err := claudecode.New("")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	out, err := c.Complete(context.Background(), "sys", "user")
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if out != "package user_test" {
		t.Errorf("Complete = %q, want %q", out, "package user_test")
	}
}

// Example test: a non-zero exit becomes an error carrying stderr.
func TestComplete_PropagatesFailure(t *testing.T) {
	t.Setenv("CUPEL_CLAUDE_BIN", fakeBin(t, `echo "boom" >&2; exit 1`))
	c, err := claudecode.New("")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := c.Complete(context.Background(), "sys", "user"); err == nil {
		t.Error("expected error on non-zero exit")
	}
}
