//go:build windows

package keylog

import (
	"github.com/sandgardenhq/affiro/asig"
	"github.com/sandgardenhq/affiro/asig/keylog/internal/windriver"
)

type monitor struct{}

func (monitor) Pop() (asig.Event, bool) {
	return windriver.Pop()
}

func (monitor) Stop() {
	windriver.Uninstall()
}

func Start() (Monitor, error) {
	if err := windriver.StartKeyMonitor(); err != nil {
		return nil, err
	}
	return monitor{}, nil
}
