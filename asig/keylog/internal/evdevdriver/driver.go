//go:build linux

package evdevdriver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/sandgardenhq/affiro/asig"
)

var eventsMu sync.Mutex
var events []asig.Event

func emit(e asig.Event) {
	eventsMu.Lock()
	events = append([]asig.Event{e}, events...)
	eventsMu.Unlock()
}

// Pop returns and removes the oldest captured event. Matches linuxdriver.Pop's shape and
// ordering exactly, so keylog_linux.go can dispatch to either backend interchangeably.
func Pop() (asig.Event, bool) {
	eventsMu.Lock()
	defer eventsMu.Unlock()
	if len(events) == 0 {
		return nil, false
	}
	ev := events[len(events)-1]
	events = events[:len(events)-1]
	return ev, true
}

// StartKeyMonitor opens every keyboard-capable device under /dev/input, starts a reader
// goroutine per device, and watches for keyboards being hot-plugged in or out. It blocks for
// the life of the process: there is no graceful shutdown path here, matching
// linuxdriver.StartKeyMonitor and cmd/affiro's existing os.Exit(1)-on-signal handling.
func StartKeyMonitor() error {
	active := map[string]*os.File{}
	var activeMu sync.Mutex

	startDevice := func(name string) {
		path := filepath.Join(defaultInputDir, name)
		f, err := os.Open(path) //nolint:gosec // path is built from a fixed base dir + a name matched against eventNodePattern.
		if err != nil {
			// Permission denied is the expected outcome for every non-keyboard device.
			if !os.IsPermission(err) {
				fmt.Println("evdev: failed to open " + path + ": " + err.Error())
			}
			return
		}

		// Read capabilities via SyscallConn rather than f.Fd(): Fd() permanently flips the
		// file to blocking mode, which would defeat the Close()-unblocks-Read mechanism
		// stopDevice relies on below to interrupt this device's reader goroutine on hot-unplug.
		rc, err := f.SyscallConn()
		if err != nil {
			fmt.Println("evdev: failed to access " + path + ": " + err.Error())
			_ = f.Close()
			return
		}

		var bitmap []byte
		var capErr error
		if err := rc.Control(func(fd uintptr) {
			bitmap, capErr = readKeyCapabilities(int(fd))
		}); err != nil {
			fmt.Println("evdev: failed to access " + path + ": " + err.Error())
			_ = f.Close()
			return
		}
		if capErr != nil {
			fmt.Println("evdev: failed to read capabilities for " + path + ": " + capErr.Error())
			_ = f.Close()
			return
		}
		if !isKeyboardCapable(bitmap) {
			_ = f.Close()
			return
		}

		activeMu.Lock()
		active[name] = f
		activeMu.Unlock()

		go func() {
			var mods modifierState
			_ = readEvents(f, &mods, emit) // returns once f is closed or hits a read error
			activeMu.Lock()
			delete(active, name)
			activeMu.Unlock()
			_ = f.Close()
		}()
	}

	stopDevice := func(name string) {
		activeMu.Lock()
		f, ok := active[name]
		delete(active, name)
		activeMu.Unlock()
		if ok {
			_ = f.Close() // unblocks that device's pending Read, ending its reader goroutine
		}
	}

	names, err := listEventNodes(defaultInputDir)
	if err != nil {
		return err
	}
	for _, name := range names {
		startDevice(name)
	}

	return watchHotplug(context.Background(), defaultInputDir, startDevice, stopDevice)
}
