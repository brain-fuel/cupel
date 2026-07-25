// Package internal holds the claudecode component implementation: invoking the
// Claude Code CLI in headless print mode.
package internal

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Client invokes the `claude` CLI.
type Client struct {
	Bin   string // path to the claude binary
	Model string // model id, or "" for the CLI default
}

// New resolves the binary and model. The binary defaults to "claude" on PATH
// and may be overridden with CUPEL_CLAUDE_BIN. The model falls back to
// CUPEL_MODEL and may be left empty to use the CLI's configured default.
func New(model string) (Client, error) {
	bin := strings.TrimSpace(os.Getenv("CUPEL_CLAUDE_BIN"))
	if bin == "" {
		bin = "claude"
	}
	path, err := exec.LookPath(bin)
	if err != nil {
		return Client{}, fmt.Errorf("claudecode: %q not found on PATH (install Claude Code, or set CUPEL_CLAUDE_BIN): %w", bin, err)
	}
	if model == "" {
		model = os.Getenv("CUPEL_MODEL")
	}
	return Client{Bin: path, Model: model}, nil
}

// ModelLabel is the model for display.
func (c Client) ModelLabel() string {
	if c.Model == "" {
		return "(claude-code default)"
	}
	return c.Model
}

// Complete runs the CLI once: the user prompt is the -p argument, the system
// prompt is appended to Claude Code's default system prompt, and plain text is
// requested on stdout.
func (c Client) Complete(ctx context.Context, system, user string) (string, error) {
	args := []string{"-p", user, "--output-format", "text"}
	if system != "" {
		args = append(args, "--append-system-prompt", system)
	}
	if c.Model != "" {
		args = append(args, "--model", c.Model)
	}

	cmd := exec.CommandContext(ctx, c.Bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("claudecode: %s failed: %s", c.Bin, msg)
	}
	out := strings.TrimSpace(stdout.String())
	if out == "" {
		return "", fmt.Errorf("claudecode: empty response from %s", c.Bin)
	}
	return out, nil
}
