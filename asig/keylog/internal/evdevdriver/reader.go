//go:build linux

package evdevdriver

import (
	"encoding/binary"
	"io"

	"github.com/sandgardenhq/affiro/asig"
)

// inputEventSize is sizeof(struct input_event) on 64-bit Linux: a 16-byte struct timeval
// (64-bit tv_sec + 64-bit tv_usec) followed by __u16 type, __u16 code, __s32 value. This
// backend assumes 64-bit Linux; 32-bit Linux uses a different (12-byte timeval) layout and is
// not supported.
const inputEventSize = 24

const (
	evSyn uint16 = 0 // EV_SYN, from <linux/input-event-codes.h>
	evKey uint16 = 1 // EV_KEY, from <linux/input-event-codes.h>
)

const (
	keyStateReleased int32 = 0
	keyStatePressed  int32 = 1
	keyStateRepeat   int32 = 2
)

// parseInputEvent decodes the type/code/value fields of one raw input_event record. asig has
// no use for the leading timestamp, so it's decoded past but discarded.
func parseInputEvent(raw [inputEventSize]byte) (evType, code uint16, value int32) {
	evType = binary.NativeEndian.Uint16(raw[16:18])
	code = binary.NativeEndian.Uint16(raw[18:20])
	value = int32(binary.NativeEndian.Uint32(raw[20:24]))
	return evType, code, value
}

// decodeKeyEvent processes one EV_KEY record against the current modifier state and returns
// the asig.Event to emit, if any. Releases only update modifier state and never emit an
// event. Presses and autorepeats emit an event for any keycode found in keycodeTable
// including the modifier keys themselves, matching the X11 driver.
func decodeKeyEvent(mods *modifierState, code uint16, value int32) (asig.Event, bool) {
	mods.apply(code, value != keyStateReleased)

	if value != keyStatePressed && value != keyStateRepeat {
		return nil, false
	}

	r, c, ok := lookupKeyCode(code, mods)
	if !ok {
		return nil, false
	}
	var s string
	if r != 0 {
		s = string(r)
	}

	return asig.KeyDownEvent{
		Key:            c,
		String:         s,
		ControlPressed: mods.controlPressed(),
		ShiftPressed:   mods.shiftPressed(),
		SpecialPressed: mods.specialPressed(),
	}, true
}

// readEvents reads input_event records from r until it errors (including io.EOF when r is
// exhausted), decoding each EV_KEY record and calling emit for every asig.Event produced. mods
// holds this device's modifier state across the whole call, since each physical keyboard
// tracks its own modifier keys independently.
func readEvents(r io.Reader, mods *modifierState, emit func(asig.Event)) error {
	var raw [inputEventSize]byte
	for {
		if _, err := io.ReadFull(r, raw[:]); err != nil {
			return err
		}

		evType, code, value := parseInputEvent(raw)
		if evType != evKey {
			continue
		}

		if ev, ok := decodeKeyEvent(mods, code, value); ok {
			emit(ev)
		}
	}
}
