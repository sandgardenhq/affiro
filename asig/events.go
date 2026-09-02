package asig

import "golang.org/x/mobile/event/key"

type Event interface {
	asigEvent()
}

type MouseUpEvent struct{}

func (MouseUpEvent) asigEvent() {}

// We ignore mouse down because mouse up is when clicks usually result in an action;
// this is less consistent than keydown/up, but unlike with keys mousedown/up are always
// paired without repeats

type KeyDownEvent struct {
	Key            key.Code
	String         string
	ControlPressed bool
	SpecialPressed bool
	ShiftPressed   bool
}

func (KeyDownEvent) asigEvent() {}

// we ignore keyup because keydown is usually when keys actually enter a typed document;
// also, holding keys will trigger keydown.keydown.keydown... without any ups.
