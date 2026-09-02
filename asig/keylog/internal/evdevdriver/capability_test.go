//go:build linux

package evdevdriver

import "testing"

func makeBitmap(codes []uint16) []byte {
	var max uint16
	for _, c := range codes {
		if c > max {
			max = c
		}
	}
	bitmap := make([]byte, max/bitsPerByte+1)
	for _, c := range codes {
		bitmap[c/bitsPerByte] |= 1 << (c % bitsPerByte)
	}
	return bitmap
}

func allBut(codes []uint16, exclude uint16) []uint16 {
	out := make([]uint16, 0, len(codes))
	for _, c := range codes {
		if c != exclude {
			out = append(out, c)
		}
	}
	return out
}

func TestIsKeyboardCapable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		set  []uint16
		want bool
	}{
		{"full keyboard", letterKeycodes, true},
		{"power button only", []uint16{116}, false},
		{"missing one letter", allBut(letterKeycodes, keyZ), false},
		{"empty bitmap", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := isKeyboardCapable(makeBitmap(tt.set)); got != tt.want {
				t.Errorf("isKeyboardCapable(%v) = %v, want %v", tt.set, got, tt.want)
			}
		})
	}
}

func TestEvdevGBitRequest(t *testing.T) {
	t.Parallel()

	// EVIOCGBIT(EV_KEY, 96) computed by hand from <linux/input.h>'s
	// #define EVIOCGBIT(ev,len) _IOC(_IOC_READ, 'E', 0x20 + (ev), len)
	// and <asm-generic/ioctl.h>'s _IOC encoding, for ev=1 (EV_KEY), len=96:
	//   dir=2, type='E'=0x45, nr=0x20+1=0x21, size=96
	//   req = dir<<30 | type<<8 | nr | size<<16
	const want = 2<<30 | 0x45<<8 | 0x21 | 96<<16

	if got := evdevGBitRequest(1, 96); got != want {
		t.Errorf("evdevGBitRequest(1, 96) = %#x, want %#x", got, want)
	}
}
