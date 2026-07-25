// Package claudecode is the public interface of the claudecode component: a
// backend that drives the local Claude Code CLI (`claude`) in headless print
// mode instead of calling the Anthropic API directly. It is an alternative to
// the claude component and satisfies the same gen.Completer contract.
//
// claudecode is a non-deterministic brick with input and output effects: it
// reads PATH/environment and shells out to a subprocess whose reply varies.
package claudecode

import (
	"context"

	"goforge.dev/cupel/components/claudecode/internal"
)

// Client drives the `claude` CLI.
type Client struct{ impl internal.Client }

// New locates the `claude` binary (overridable via CUPEL_CLAUDE_BIN) and
// resolves the model. An empty model means the CLI's own default; model also
// falls back to CUPEL_MODEL.
func New(model string) (Client, error) {
	impl, err := internal.New(model)
	return Client{impl: impl}, err
}

// Model reports the model the CLI will be asked to use, or "(claude-code
// default)" when none is configured.
func (c Client) Model() string { return c.impl.ModelLabel() }

// Complete runs `claude -p` with the system prompt appended, returning stdout.
func (c Client) Complete(ctx context.Context, system, user string) (string, error) {
	return c.impl.Complete(ctx, system, user)
}
