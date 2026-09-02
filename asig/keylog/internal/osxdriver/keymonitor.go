//go:build darwin

package osxdriver

/*
#include <stdbool.h>
#include "keymonitor.h"
#cgo LDFLAGS: -framework AppKit
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/sandgardenhq/affiro/asig"
)

func StartKeyMonitor() {
	C.Start_Key_Monitor()
}

type Monitor struct {
	monitor unsafe.Pointer
}

func Pop() (asig.Event, bool) {
	eData := C.Global_Key_Monitor_Pop()
	if KeyCode(eData.keyCode) == KeyCodeNoEvent {
		return nil, false
	}
	switch eventType(eData._type) {
	case keyDownEventType:
		rn := C.GoString(eData.characters)
		return asig.KeyDownEvent{
			Key:            convertKeyCode(KeyCode(eData.keyCode)),
			String:         rn,
			ControlPressed: eData.modifierFlags&ctrlMask == ctrlMask,
			SpecialPressed: eData.modifierFlags&cmdMask == cmdMask,
			ShiftPressed:   eData.modifierFlags&shiftMask == shiftMask,
		}, true
	case leftMouseUpEventType, rightMouseUpEventType:
		return asig.MouseUpEvent{}, true
	default:
		// This is dead code until the code around it changes;
		// we only subscribe to mouse and key events
		fmt.Println("unknown event type", eData._type)
		return nil, false
	}
}

const (
	shiftMask  = 131330
	cmdMask    = 262401
	optionMask = 524576
	ctrlMask   = 1048840
)

type eventType int

const (
	// Raw AppKit NSEventType values (see NSEvent.h); wait_for_events in
	// keymonitor.m only subscribes to left/right mouse-up and key-down.
	leftMouseUpEventType  eventType = 2
	rightMouseUpEventType eventType = 4
	keyDownEventType      eventType = 10
)
