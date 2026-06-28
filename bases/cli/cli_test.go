package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestHelpAliases(t *testing.T) {
	for _, arg := range []string{":help", "--help", "-h", "help"} {
		var stdout, stderr bytes.Buffer
		if code := run([]string{arg}, strings.NewReader(""), &stdout, &stderr); code != 0 {
			t.Fatalf("%s exit code = %d, stderr: %s", arg, code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "Usage:") {
			t.Fatalf("%s help output missing usage: %q", arg, stdout.String())
		}
	}
}

func TestVersionAliases(t *testing.T) {
	for _, arg := range []string{":version", "--version", "-v", "version"} {
		var stdout, stderr bytes.Buffer
		if code := run([]string{arg}, strings.NewReader(""), &stdout, &stderr); code != 0 {
			t.Fatalf("%s exit code = %d, stderr: %s", arg, code, stderr.String())
		}
		if !strings.HasPrefix(stdout.String(), "cupel ") {
			t.Fatalf("%s version output = %q", arg, stdout.String())
		}
	}
}

func TestRejectsUnexpectedArgument(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"name:user", "specification"}, strings.NewReader(""), &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), `unexpected argument "specification"`) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRejectsUnknownOption(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"name:user", "spce:typo"}, strings.NewReader(""), &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), `unknown argument "spce:"`) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
