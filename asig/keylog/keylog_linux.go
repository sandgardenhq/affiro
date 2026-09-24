//go:build linux

package keylog

import (
	"fmt"
	"os"

	"github.com/sandgardenhq/affiro/asig"
	"github.com/sandgardenhq/affiro/asig/keylog/internal/evdevdriver"
	"github.com/sandgardenhq/affiro/asig/keylog/internal/x11driver"
)

type keylogBackend struct {
	start func() error
	pop   func() (asig.Event, bool)
	name  string
}

var x11Backend = keylogBackend{name: "x11", start: x11driver.StartKeyMonitor, pop: x11driver.Pop}
var evdevBackend = keylogBackend{name: "evdev", start: evdevdriver.StartKeyMonitor, pop: evdevdriver.Pop}

// selectBackend chooses which Linux capture backend to use, given the AFFIRO_KEYLOG_BACKEND
// override and the XDG_SESSION_TYPE the desktop session reports. override, if "evdev" or
// "x11", always wins (useful for testing, and for edge cases like an XWayland session where a
// user wants to force one backend or the other). Otherwise, a Wayland session ("wayland")
// selects evdev; anything else (X11, or unset, which covers most non-desktop contexts)
// keeps the existing X11 backend.
func selectBackend(override, sessionType string) keylogBackend {
	switch override {
	case "evdev":
		return evdevBackend
	case "x11":
		return x11Backend
	}
	if sessionType == "wayland" {
		return evdevBackend
	}
	return x11Backend
}

func NewMonitor() Monitor {
	backend := selectBackend(os.Getenv("AFFIRO_KEYLOG_BACKEND"), os.Getenv("XDG_SESSION_TYPE"))
	return monitor{
		backend: backend,
	}
}

type monitor struct {
	backend keylogBackend
}

func (m monitor) Pop() (asig.Event, bool) {
	return m.backend.pop()
}

func (monitor) Stop() {
	// NOP
}

func (m monitor) Start() error {
	go func() {
		if err := m.backend.start(); err != nil {
			fmt.Println("failed to start key monitor: " + err.Error())
		}
	}()
	return nil
}
