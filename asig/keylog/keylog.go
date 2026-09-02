package keylog

import "github.com/sandgardenhq/affiro/asig"

// Monitor represents a running keystroke-capture session returned by Start.
// Pop and Stop are safe to call from goroutines other than the one that
// called Start.
type Monitor interface {
	// Pop returns and removes the oldest captured event, if any is queued.
	Pop() (asig.Event, bool)
	// Stop ends capture. On backends with no shutdown hook, it is a no-op.
	Stop()
}
