//go:build darwin

package keylog

import (
	"github.com/sandgardenhq/affiro/asig"
	"github.com/sandgardenhq/affiro/asig/keylog/internal/osxdriver"
)

func NewMonitor() Monitor {
	return monitor{}
}

type monitor struct{}

func (monitor) Pop() (asig.Event, bool) {
	return osxdriver.Pop()
}

func (monitor) Stop() {
	// NOP
}

// Start has two possible modes:
// if another cocoa app is running with this process, it will set up its hooks and then return.
// But if no cocoa app is running, it will block forever.
func (monitor) Start() error {
	osxdriver.StartKeyMonitor()
	return nil
}
