package asig

import "golang.org/x/mobile/event/key"

const (
	// pastedBit marks a window in which the user pasted rather than typed.
	pastedBit = 128
	// maxCount is as high as a window's event counter goes before it saturates; the paste
	// bit sits above it.
	maxCount = 127
)

// applyEvent folds one event into the byte standing for the window it landed in. Both
// algorithm versions score events identically; only their encoding of a pause differs.
func applyEvent(b byte, ev Event) byte {
	switch v := ev.(type) {
	case KeyDownEvent:
		return applyKeyDown(b, v)
	case MouseUpEvent:
		return applyMouseUp(b)
	}
	return b
}

// applyKeyDown counts a keystroke, or sets the paste bit when the keystroke was a paste.
func applyKeyDown(b byte, ev KeyDownEvent) byte {
	if (ev.SpecialPressed || ev.ControlPressed) && ev.Key == key.CodeV {
		return b | pastedBit
	}
	if b == maxCount || b == maxCount|pastedBit {
		return b
	}
	return b + 1
}

// applyMouseUp counts a mouse release as two, leaving the counter alone when two more would
// carry into the paste bit.
func applyMouseUp(b byte) byte {
	if b == maxCount || b == maxCount-1 || b == maxCount|pastedBit || b == (maxCount|pastedBit)-1 {
		return b
	}
	return b + 2
}
