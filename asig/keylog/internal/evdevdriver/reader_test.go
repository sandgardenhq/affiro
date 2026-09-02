//go:build linux

package evdevdriver

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"
	"testing/iotest"

	"github.com/sandgardenhq/affiro/asig"
	"golang.org/x/mobile/event/key"
)

func rawEvent(evType, code uint16, value int32) []byte {
	buf := make([]byte, inputEventSize)
	binary.NativeEndian.PutUint16(buf[16:18], evType)
	binary.NativeEndian.PutUint16(buf[18:20], code)
	binary.NativeEndian.PutUint32(buf[20:24], uint32(value))
	return buf
}

func TestReadEvents(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	buf.Write(rawEvent(evSyn, 0, 0)) // EV_SYN: must be ignored, it isn't EV_KEY
	buf.Write(rawEvent(evKey, keyLeftShift, 1))
	buf.Write(rawEvent(evKey, keyA, 1))
	buf.Write(rawEvent(evKey, keyA, 0))
	buf.Write(rawEvent(evKey, keyLeftShift, 0))

	var mods modifierState
	var got []asig.Event
	err := readEvents(&buf, &mods, func(e asig.Event) { got = append(got, e) })
	if !errors.Is(err, io.EOF) {
		t.Fatalf("readEvents error = %v, want io.EOF", err)
	}

	want := []asig.Event{
		asig.KeyDownEvent{Key: key.CodeLeftShift, ShiftPressed: true},
		asig.KeyDownEvent{Key: key.CodeA, ShiftPressed: true, String: "A"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d events, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("event %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestReadEvents_Autorepeat(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	buf.Write(rawEvent(evKey, keyA, 1)) // press
	buf.Write(rawEvent(evKey, keyA, 2)) // autorepeat
	buf.Write(rawEvent(evKey, keyA, 0)) // release

	var mods modifierState
	var got []asig.Event
	err := readEvents(&buf, &mods, func(e asig.Event) { got = append(got, e) })
	if !errors.Is(err, io.EOF) {
		t.Fatalf("readEvents error = %v, want io.EOF", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d events for press+autorepeat+release, want 2 (release emits nothing): %+v", len(got), got)
	}
}

func TestReadEvents_UnknownKeycodeIgnored(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	buf.Write(rawEvent(evKey, 0xFFFF, 1))

	var mods modifierState
	var got []asig.Event
	err := readEvents(&buf, &mods, func(e asig.Event) { got = append(got, e) })
	if !errors.Is(err, io.EOF) {
		t.Fatalf("readEvents error = %v, want io.EOF", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %d events for an unmapped keycode, want 0: %+v", len(got), got)
	}
}

func TestReadEvents_PropagatesReadError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("boom")
	var mods modifierState
	err := readEvents(iotest.ErrReader(wantErr), &mods, func(asig.Event) {})
	if !errors.Is(err, wantErr) {
		t.Fatalf("readEvents error = %v, want %v", err, wantErr)
	}
}
