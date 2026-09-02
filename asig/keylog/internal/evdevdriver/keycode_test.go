//go:build linux

package evdevdriver

import (
	"testing"

	"golang.org/x/mobile/event/key"
)

func TestLookupKeyCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		code     uint16
		want     key.Code
		mods     modifierState
		wantRune rune
	}{
		{name: "letterA", code: keyA, want: key.CodeA, wantRune: 'a'},
		{name: "letterAShift", code: keyA, want: key.CodeA, wantRune: 'A', mods: modifierState{leftShift: true}},
		{name: "digit1", code: key1, want: key.Code1, wantRune: '1'},
		{name: "space", code: keySpace, want: key.CodeSpacebar, wantRune: ' '},
		{name: "leftShift", code: keyLeftShift, want: key.CodeLeftShift},
		{name: "rightMeta", code: keyRightMeta, want: key.CodeRightGUI},
		{name: "enter", code: keyEnter, want: key.CodeReturnEnter},
		{name: "upArrow", code: keyUp, want: key.CodeUpArrow},
		{name: "keypad5", code: keyKP5, want: key.CodeKeypad5, wantRune: '5'},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rn, got, ok := lookupKeyCode(tt.code, &tt.mods)
			if !ok {
				t.Fatalf("lookupKeyCode(%d) returned ok=false, want true", tt.code)
			}
			if got != tt.want {
				t.Errorf("lookupKeyCode(%d) = %v, want %v", tt.code, got, tt.want)
			}
			if rn != tt.wantRune {
				t.Errorf("lookupKeyCode(%d) = %v, want %v", tt.code, got, tt.want)
			}
		})
	}
}

func TestLookupKeyCode_Unknown(t *testing.T) {
	t.Parallel()

	const unknownCode = 0xFFFF
	if _, _, ok := lookupKeyCode(unknownCode, &modifierState{}); ok {
		t.Fatalf("lookupKeyCode(%d) returned ok=true, want false for an unmapped code", unknownCode)
	}
}
