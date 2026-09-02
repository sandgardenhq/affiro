//go:build linux

package evdevdriver

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"unsafe"

	"golang.org/x/sys/unix"
)

const defaultInputDir = "/dev/input"

var eventNodePattern = regexp.MustCompile(`^event[0-9]+$`)

// listEventNodes returns the names (not full paths) of every eventN device node directly
// inside dir. It does not classify them as keyboards. Callers open and probe each one via
// readKeyCapabilities/isKeyboardCapable.
func listEventNodes(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}

	var names []string
	for _, entry := range entries {
		if eventNodePattern.MatchString(entry.Name()) {
			names = append(names, entry.Name())
		}
	}
	return names, nil
}

// watchHotplug watches dir for created and removed entries whose names match
// eventNodePattern, invoking onAdd/onRemove with the entry name (not full path) for each. It
// blocks until ctx is done (returning ctx.Err()) or the underlying inotify watch fails.
func watchHotplug(ctx context.Context, dir string, onAdd, onRemove func(name string)) error {
	fd, err := unix.InotifyInit1(unix.IN_CLOEXEC | unix.IN_NONBLOCK)
	if err != nil {
		return fmt.Errorf("inotify_init1: %w", err)
	}
	defer func() { _ = unix.Close(fd) }()

	if _, err := unix.InotifyAddWatch(fd, dir, unix.IN_CREATE|unix.IN_DELETE); err != nil {
		return fmt.Errorf("inotify_add_watch %s: %w", dir, err)
	}

	const pollTimeoutMillis = 200

	pollFds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
	buf := make([]byte, 4096)

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		n, err := unix.Poll(pollFds, pollTimeoutMillis)
		if err != nil {
			if errors.Is(err, unix.EINTR) {
				continue
			}
			return fmt.Errorf("polling inotify fd: %w", err)
		}
		if n == 0 {
			continue // timed out with nothing ready; loop back to re-check ctx
		}

		nRead, err := unix.Read(fd, buf)
		if err != nil {
			if errors.Is(err, unix.EAGAIN) {
				continue
			}
			return fmt.Errorf("reading inotify events: %w", err)
		}

		offset := 0
		for offset+unix.SizeofInotifyEvent <= nRead {
			raw := (*unix.InotifyEvent)(unsafe.Pointer(&buf[offset]))
			nameStart := offset + unix.SizeofInotifyEvent
			nameEnd := nameStart + int(raw.Len)
			if nameEnd > nRead {
				break
			}
			name := string(bytes.TrimRight(buf[nameStart:nameEnd], "\x00"))
			offset = nameEnd

			if !eventNodePattern.MatchString(name) {
				continue
			}
			switch {
			case raw.Mask&unix.IN_CREATE != 0:
				onAdd(name)
			case raw.Mask&unix.IN_DELETE != 0:
				onRemove(name)
			}
		}
	}
}
