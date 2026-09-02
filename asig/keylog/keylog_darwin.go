//go:build darwin

package keylog

import (
	"github.com/sandgardenhq/affiro/asig"
	"github.com/sandgardenhq/affiro/asig/keylog/internal/osxdriver"
)

type monitor struct{}

func (monitor) Pop() (asig.Event, bool) {
	return osxdriver.Pop()
}

func (monitor) Stop() {
	// osxdriver exposes no shutdown hook today; matches the previous
	// StopKeyMonitor's nop.
}

// Start begins capturing in the background and returns immediately.
// osxdriver.StartKeyMonitor blocks for the life of the process and has no
// error return, so there is nothing to report if it fails internally.
func Start() (Monitor, error) {
	osxdriver.StartKeyMonitor()
	return monitor{}, nil
}
