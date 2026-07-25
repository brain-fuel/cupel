// Package claude is the public interface of the claude component: a thin client
// for the Anthropic Messages API used to turn a prompt into generated tests.
//
// claude is a non-deterministic brick with input and output effects: it reads
// configuration from the environment and sends a request over the network whose
// reply varies between calls.
package claude

import (
	"context"

	"goforge.dev/cupel/components/claude/internal"
)

// DefaultModel is used when neither the model argument nor CUPEL_MODEL is set.
const DefaultModel = internal.DefaultModel

// Client talks to the Anthropic Messages API.
type Client struct{ impl internal.Client }

// New builds a Client. The API key is read from ANTHROPIC_API_KEY; model falls
// back to CUPEL_MODEL and then DefaultModel when empty.
func New(model string) (Client, error) {
	impl, err := internal.New(model)
	return Client{impl: impl}, err
}

// Model reports the model the client will call.
func (c Client) Model() string { return c.impl.Model }

// Complete sends system + user and returns the concatenated text of the reply.
// The system prompt is sent with prompt caching enabled.
func (c Client) Complete(ctx context.Context, system, user string) (string, error) {
	return c.impl.Complete(ctx, system, user)
}
