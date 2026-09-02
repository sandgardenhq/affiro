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
	name  string
	start func() error
	pop   func() (asig.Event, bool)
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

type monitor struct {
	backend keylogBackend
}

func (m monitor) Pop() (asig.Event, bool) {
	return m.backend.pop()
}

func (monitor) Stop() {
	// Neither the x11 nor evdev backend supports graceful shutdown today;
}

// Start picks a backend (see selectBackend) and begins capturing in the
// background, returning immediately. The chosen backend's start function
// blocks for the life of the process, so any error it returns after this
// point is only logged, not returned — callers that need the Monitor to
// have started successfully cannot observe that here.
func Start() (Monitor, error) {
	backend := selectBackend(os.Getenv("AFFIRO_KEYLOG_BACKEND"), os.Getenv("XDG_SESSION_TYPE"))
	go func() {
		if err := backend.start(); err != nil {
			fmt.Println("failed to start key monitor: " + err.Error())
		}
	}()
	return monitor{backend: backend}, nil
}
