//go:build linux

package evdevdriver

import "testing"

func TestModifierState(t *testing.T) {
	t.Parallel()

	var m modifierState

	m.apply(keyLeftShift, true)
	if !m.shiftPressed() {
		t.Fatal("expected shiftPressed=true after left shift press")
	}

	m.apply(keyLeftShift, false)
	if m.shiftPressed() {
		t.Fatal("expected shiftPressed=false after left shift release")
	}

	m.apply(keyRightCtrl, true)
	if !m.controlPressed() {
		t.Fatal("expected controlPressed=true after right ctrl press")
	}

	m.apply(keyLeftMeta, true)
	if !m.specialPressed() {
		t.Fatal("expected specialPressed=true after left meta press")
	}

	m.apply(keyA, true)
	if !m.controlPressed() || !m.specialPressed() {
		t.Fatal("pressing a non-modifier key must not clear existing modifier state")
	}
	if m.shiftPressed() {
		t.Fatal("pressing a non-modifier key must not set shift")
	}
}

func TestModifierState_IndependentLeftRight(t *testing.T) {
	t.Parallel()

	var m modifierState
	m.apply(keyLeftShift, true)
	m.apply(keyRightShift, true)
	m.apply(keyLeftShift, false)
	if !m.shiftPressed() {
		t.Fatal("releasing left shift while right shift is still held should keep shiftPressed=true")
	}
}
