//go:build linux

package keylog

import (
	"testing"

	"github.com/sandgardenhq/affiro/asig"
)

func TestSelectBackend(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		override    string
		sessionType string
		want        string
	}{
		{"wayland session picks evdev", "", "wayland", "evdev"},
		{"x11 session picks x11", "", "x11", "x11"},
		{"unset session type picks x11", "", "", "x11"},
		{"override evdev wins over x11 session", "evdev", "x11", "evdev"},
		{"override x11 wins over wayland session", "x11", "wayland", "x11"},
		{"unrecognized override falls through to session type", "bogus", "wayland", "evdev"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := selectBackend(tt.override, tt.sessionType)
			if got.name != tt.want {
				t.Errorf("selectBackend(%q, %q) = %q, want %q", tt.override, tt.sessionType, got.name, tt.want)
			}
		})
	}
}

func TestMonitor_Pop(t *testing.T) {
	t.Parallel()

	want := asig.KeyDownEvent{Key: 42}
	m := monitor{backend: keylogBackend{
		name: "fake",
		pop:  func() (asig.Event, bool) { return want, true },
	}}

	got, ok := m.Pop()
	if !ok || got != want {
		t.Errorf("Pop() = (%v, %v), want (%v, true)", got, ok, want)
	}
}

func TestMonitor_Stop(t *testing.T) {
	t.Parallel()

	m := monitor{backend: keylogBackend{name: "fake"}}
	m.Stop() // must not panic; linux has no shutdown hook to call
}
