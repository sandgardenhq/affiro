//go:build linux

package evdevdriver

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestListEventNodes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	for _, name := range []string{"event0", "event12", "mouse0", "js0", "eventfoo"} {
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("creating fixture file %s: %v", name, err)
		}
		if err := f.Close(); err != nil {
			t.Fatalf("closing fixture file %s: %v", name, err)
		}
	}

	got, err := listEventNodes(dir)
	if err != nil {
		t.Fatalf("listEventNodes: %v", err)
	}

	want := map[string]bool{"event0": true, "event12": true}
	if len(got) != len(want) {
		t.Fatalf("listEventNodes = %v, want exactly the entries in %v", got, want)
	}
	for _, name := range got {
		if !want[name] {
			t.Errorf("listEventNodes returned unexpected entry %q", name)
		}
	}
}

func TestListEventNodes_MissingDir(t *testing.T) {
	t.Parallel()

	if _, err := listEventNodes(filepath.Join(t.TempDir(), "does-not-exist")); err == nil {
		t.Fatal("expected an error for a missing directory")
	}
}

func TestWatchHotplug(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	var added, removed []string

	done := make(chan error, 1)
	go func() {
		done <- watchHotplug(ctx, dir,
			func(name string) { mu.Lock(); added = append(added, name); mu.Unlock() },
			func(name string) { mu.Lock(); removed = append(removed, name); mu.Unlock() },
		)
	}()

	// Give the watch goroutine a moment to register its inotify watch before creating files.
	time.Sleep(50 * time.Millisecond)

	path := filepath.Join(dir, "event7")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("creating fixture file: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("closing fixture file: %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatalf("removing fixture file: %v", err)
	}

	other := filepath.Join(dir, "mouse0") // non-matching name, must be ignored
	f, err = os.Create(other)
	if err != nil {
		t.Fatalf("creating fixture file: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("closing fixture file: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		mu.Lock()
		gotBoth := len(added) > 0 && len(removed) > 0
		mu.Unlock()
		if gotBoth {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for hotplug events, added=%v removed=%v", added, removed)
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	if err := <-done; err == nil {
		t.Fatal("expected watchHotplug to return an error once its context is canceled")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(added) != 1 || added[0] != "event7" {
		t.Errorf("added = %v, want [event7]", added)
	}
	if len(removed) != 1 || removed[0] != "event7" {
		t.Errorf("removed = %v, want [event7]", removed)
	}
}
