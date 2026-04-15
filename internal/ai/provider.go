package ai

import "context"

// AIProvider defines the interface for AI backends.
// All implementations MUST support streaming output.
type AIProvider interface {
	Stream(ctx context.Context, system, user string) (<-chan string, <-chan error)
}
