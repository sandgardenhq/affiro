//go:build linux

package evdevdriver

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/unix"
)

const bitsPerByte = 8

// letterKeycodes are the 26 alphabetic keycodes. A device is classified as a keyboard only if
// it reports all of them as supported: that reliably distinguishes a full keyboard from
// single/few-key input devices (power buttons, remote controls, etc.) without depending on
// udev's ID_INPUT_KEYBOARD property, so classification works identically with or without udev
// running.
var letterKeycodes = []uint16{
	keyA, keyB, keyC, keyD, keyE, keyF, keyG, keyH, keyI, keyJ, keyK, keyL, keyM,
	keyN, keyO, keyP, keyQ, keyR, keyS, keyT, keyU, keyV, keyW, keyX, keyY, keyZ,
}

// bitSet reports whether keycode's bit is set in an EVIOCGBIT(EV_KEY, ...) capability bitmap.
func bitSet(bitmap []byte, keycode uint16) bool {
	byteIndex := int(keycode) / bitsPerByte
	if byteIndex >= len(bitmap) {
		return false
	}
	return bitmap[byteIndex]&(1<<(int(keycode)%bitsPerByte)) != 0
}

// isKeyboardCapable classifies an EV_KEY capability bitmap (as returned by
// readKeyCapabilities) as belonging to a full keyboard.
func isKeyboardCapable(bitmap []byte) bool {
	for _, code := range letterKeycodes {
		if !bitSet(bitmap, code) {
			return false
		}
	}
	return true
}

// Linux ioctl request encoding, from <asm-generic/ioctl.h>.
const (
	iocNRBits   = 8
	iocTypeBits = 8
	iocSizeBits = 14

	iocNRShift   = 0
	iocTypeShift = iocNRShift + iocNRBits
	iocSizeShift = iocTypeShift + iocTypeBits
	iocDirShift  = iocSizeShift + iocSizeBits

	iocRead = 2

	evdevIOCType  = 'E'
	evdevGBitBase = 0x20 // EVIOCGBIT(ev, len) = _IOC(_IOC_READ, 'E', 0x20 + ev, len)
)

func evdevGBitRequest(ev, length int) uintptr {
	return uintptr(iocRead<<iocDirShift | evdevIOCType<<iocTypeShift | (evdevGBitBase+ev)<<iocNRShift | length<<iocSizeShift)
}

// keyCapabilityBitmapSize covers keycodes 0 through KEY_MAX (0x2ff), the largest keycode
// evdev's stable ABI defines.
const keyCapabilityBitmapSize = (0x2ff / bitsPerByte) + 1

// readKeyCapabilities returns the EV_KEY capability bitmap for an already-open device fd
func readKeyCapabilities(fd int) ([]byte, error) {
	bitmap := make([]byte, keyCapabilityBitmapSize)
	req := evdevGBitRequest(unix.EV_KEY, len(bitmap))
	//nolint:gosec // fd and the bitmap buffer are both ours; this is the standard EVIOCGBIT pattern.
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), req, uintptr(unsafe.Pointer(&bitmap[0])))
	if errno != 0 {
		return nil, fmt.Errorf("EVIOCGBIT ioctl: %w", errno)
	}
	return bitmap, nil
}
