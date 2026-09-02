//go:build linux

package evdevdriver

// modifierState tracks which of shift/control/meta are currently held down, driven directly
// by those keys' own press/release records (evdev has no separate "modifiers changed" event
// type).
//
// Alt is deliberately not tracked here: it flows through as an ordinary key via keycodeTable's
// keyLeftAlt/keyRightAlt entries, matching the existing X11 driver
// (asig/keylog/internal/linuxdriver/driver.go), which likewise never sets a modifier flag for
// Alt. Keeping that identical across backends matters because a captured signature shouldn't
// depend on which display server produced it.
type modifierState struct {
	leftShift, rightShift     bool
	leftControl, rightControl bool
	leftMeta, rightMeta       bool
}

// apply updates the modifier state for a press (pressed=true) or release (pressed=false) of
// the given keycode. Keycodes that aren't one of the tracked modifier keys are ignored.
func (m *modifierState) apply(code uint16, pressed bool) {
	switch code {
	case keyLeftShift:
		m.leftShift = pressed
	case keyRightShift:
		m.rightShift = pressed
	case keyLeftCtrl:
		m.leftControl = pressed
	case keyRightCtrl:
		m.rightControl = pressed
	case keyLeftMeta:
		m.leftMeta = pressed
	case keyRightMeta:
		m.rightMeta = pressed
	}
}

func (m *modifierState) shiftPressed() bool   { return m.leftShift || m.rightShift }
func (m *modifierState) controlPressed() bool { return m.leftControl || m.rightControl }
func (m *modifierState) specialPressed() bool { return m.leftMeta || m.rightMeta }
