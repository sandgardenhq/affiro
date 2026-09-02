//go:build linux

// Package evdevdriver captures keystrokes on Wayland Linux sessions by reading raw evdev
// (/dev/input/event*) records directly, since Wayland's security model gives no application
// a way to observe another's input events.
package evdevdriver

import (
	"unicode"

	"golang.org/x/mobile/event/key"
)

// Linux kernel keycodes, from <linux/input-event-codes.h>. These numbers are part of the
// stable kernel input ABI and do not change across kernel versions.
const (
	keyEsc        = 1
	key1          = 2
	key2          = 3
	key3          = 4
	key4          = 5
	key5          = 6
	key6          = 7
	key7          = 8
	key8          = 9
	key9          = 10
	key0          = 11
	keyMinus      = 12
	keyEqual      = 13
	keyBackspace  = 14
	keyTab        = 15
	keyQ          = 16
	keyW          = 17
	keyE          = 18
	keyR          = 19
	keyT          = 20
	keyY          = 21
	keyU          = 22
	keyI          = 23
	keyO          = 24
	keyP          = 25
	keyLeftBrace  = 26
	keyRightBrace = 27
	keyEnter      = 28
	keyLeftCtrl   = 29
	keyA          = 30
	keyS          = 31
	keyD          = 32
	keyF          = 33
	keyG          = 34
	keyH          = 35
	keyJ          = 36
	keyK          = 37
	keyL          = 38
	keySemicolon  = 39
	keyApostrophe = 40
	keyGrave      = 41
	keyLeftShift  = 42
	keyBackslash  = 43
	keyZ          = 44
	keyX          = 45
	keyC          = 46
	keyV          = 47
	keyB          = 48
	keyN          = 49
	keyM          = 50
	keyComma      = 51
	keyDot        = 52
	keySlash      = 53
	keyRightShift = 54
	keyKPAsterisk = 55
	keyLeftAlt    = 56
	keySpace      = 57
	keyCapsLock   = 58
	keyF1         = 59
	keyF2         = 60
	keyF3         = 61
	keyF4         = 62
	keyF5         = 63
	keyF6         = 64
	keyF7         = 65
	keyF8         = 66
	keyF9         = 67
	keyF10        = 68
	keyNumLock    = 69
	keyKP7        = 71
	keyKP8        = 72
	keyKP9        = 73
	keyKPMinus    = 74
	keyKP4        = 75
	keyKP5        = 76
	keyKP6        = 77
	keyKPPlus     = 78
	keyKP1        = 79
	keyKP2        = 80
	keyKP3        = 81
	keyKP0        = 82
	keyKPDot      = 83
	keyF11        = 87
	keyF12        = 88
	keyKPEnter    = 96
	keyRightCtrl  = 97
	keyKPSlash    = 98
	keyRightAlt   = 100
	keyHome       = 102
	keyUp         = 103
	keyPageUp     = 104
	keyLeft       = 105
	keyRight      = 106
	keyEnd        = 107
	keyDown       = 108
	keyPageDown   = 109
	keyInsert     = 110
	keyDelete     = 111
	keyMute       = 113
	keyVolumeDown = 114
	keyVolumeUp   = 115
	keyKPEqual    = 117
	keyLeftMeta   = 125
	keyRightMeta  = 126
	keyCompose    = 127
)

// keycodeTable maps Linux evdev keycodes directly to key.Code. Unlike the X11 driver's
// keysym-based lookup (asig/keylog/internal/linuxdriver/x11key), this needs no layout
// resolution: key.Code already represents a physical key position, which is all asig needs
// (see asig/events.go).
var keycodeTable = map[uint16]key.Code{
	key1: key.Code1, key2: key.Code2, key3: key.Code3, key4: key.Code4, key5: key.Code5,
	key6: key.Code6, key7: key.Code7, key8: key.Code8, key9: key.Code9, key0: key.Code0,

	keyA: key.CodeA, keyB: key.CodeB, keyC: key.CodeC, keyD: key.CodeD, keyE: key.CodeE,
	keyF: key.CodeF, keyG: key.CodeG, keyH: key.CodeH, keyI: key.CodeI, keyJ: key.CodeJ,
	keyK: key.CodeK, keyL: key.CodeL, keyM: key.CodeM, keyN: key.CodeN, keyO: key.CodeO,
	keyP: key.CodeP, keyQ: key.CodeQ, keyR: key.CodeR, keyS: key.CodeS, keyT: key.CodeT,
	keyU: key.CodeU, keyV: key.CodeV, keyW: key.CodeW, keyX: key.CodeX, keyY: key.CodeY,
	keyZ: key.CodeZ,

	keyMinus: key.CodeHyphenMinus, keyEqual: key.CodeEqualSign,
	keyLeftBrace: key.CodeLeftSquareBracket, keyRightBrace: key.CodeRightSquareBracket,
	keyBackslash: key.CodeBackslash, keySemicolon: key.CodeSemicolon,
	keyApostrophe: key.CodeApostrophe, keyGrave: key.CodeGraveAccent,
	keyComma: key.CodeComma, keyDot: key.CodeFullStop, keySlash: key.CodeSlash,
	keySpace: key.CodeSpacebar,

	keyEsc: key.CodeEscape, keyTab: key.CodeTab, keyEnter: key.CodeReturnEnter,
	keyBackspace: key.CodeDeleteBackspace, keyCapsLock: key.CodeCapsLock,

	keyF1: key.CodeF1, keyF2: key.CodeF2, keyF3: key.CodeF3, keyF4: key.CodeF4,
	keyF5: key.CodeF5, keyF6: key.CodeF6, keyF7: key.CodeF7, keyF8: key.CodeF8,
	keyF9: key.CodeF9, keyF10: key.CodeF10, keyF11: key.CodeF11, keyF12: key.CodeF12,

	keyLeftShift: key.CodeLeftShift, keyRightShift: key.CodeRightShift,
	keyLeftCtrl: key.CodeLeftControl, keyRightCtrl: key.CodeRightControl,
	keyLeftAlt: key.CodeLeftAlt, keyRightAlt: key.CodeRightAlt,
	keyLeftMeta: key.CodeLeftGUI, keyRightMeta: key.CodeRightGUI,
	keyCompose: key.CodeCompose,

	keyHome: key.CodeHome, keyEnd: key.CodeEnd, keyUp: key.CodeUpArrow,
	keyDown: key.CodeDownArrow, keyLeft: key.CodeLeftArrow, keyRight: key.CodeRightArrow,
	keyPageUp: key.CodePageUp, keyPageDown: key.CodePageDown,
	keyInsert: key.CodeInsert, keyDelete: key.CodeDeleteForward,

	keyNumLock: key.CodeKeypadNumLock, keyKP0: key.CodeKeypad0, keyKP1: key.CodeKeypad1,
	keyKP2: key.CodeKeypad2, keyKP3: key.CodeKeypad3, keyKP4: key.CodeKeypad4,
	keyKP5: key.CodeKeypad5, keyKP6: key.CodeKeypad6, keyKP7: key.CodeKeypad7,
	keyKP8: key.CodeKeypad8, keyKP9: key.CodeKeypad9,
	keyKPDot: key.CodeKeypadFullStop, keyKPPlus: key.CodeKeypadPlusSign,
	keyKPMinus: key.CodeKeypadHyphenMinus, keyKPAsterisk: key.CodeKeypadAsterisk,
	keyKPSlash: key.CodeKeypadSlash, keyKPEnter: key.CodeKeypadEnter,
	keyKPEqual: key.CodeKeypadEqualSign,

	keyVolumeUp: key.CodeVolumeUp, keyVolumeDown: key.CodeVolumeDown, keyMute: key.CodeMute,
}

// lookupKeyCode returns the key.Code for a Linux evdev keycode, and false if the keycode isn't
// in the static table. An unmapped keycode is not an error -- callers should just ignore it
func lookupKeyCode(code uint16, mods *modifierState) (rune, key.Code, bool) {
	c, ok := keycodeTable[code]
	if !ok {
		return 0, 0, false
	}
	// TODO: track the current device's keymap (this optimistically assumes this has already been done
	// for us, which is unlikely)
	// in theory, when you connect to a keyboard device one of the first messages it sends is its keymap;
	// otherwise most solutions to this assume some kind of X integration; its possible wayland uses some
	// X components for keymapping still.
	var r rune
	ks, rok := simpleCodeRunes[c]
	if rok {
		r = ks.val(mods.shiftPressed())
	}
	return r, c, ok
}

func unshifted(r rune) shiftKeySwitch {
	return shiftKeySwitch{r, r}
}

func shifted(r1, r2 rune) shiftKeySwitch {
	return shiftKeySwitch{r1, r2}
}

func uppercase(r1 rune) shiftKeySwitch {
	r2 := unicode.ToUpper(r1)
	return shifted(r1, r2)
}

type shiftKeySwitch [2]rune

func (s shiftKeySwitch) val(shifted bool) rune {
	if shifted {
		return s[1]
	}
	return s[0]
}

var simpleCodeRunes = map[key.Code]shiftKeySwitch{
	key.CodeKeypad0:           unshifted('0'),
	key.CodeKeypad1:           unshifted('1'),
	key.CodeKeypad2:           unshifted('2'),
	key.CodeKeypad3:           unshifted('3'),
	key.CodeKeypad4:           unshifted('4'),
	key.CodeKeypad5:           unshifted('5'),
	key.CodeKeypad6:           unshifted('6'),
	key.CodeKeypad7:           unshifted('7'),
	key.CodeKeypad8:           unshifted('8'),
	key.CodeKeypad9:           unshifted('9'),
	key.CodeKeypadAsterisk:    unshifted('*'),
	key.CodeKeypadEqualSign:   unshifted('='),
	key.CodeKeypadFullStop:    unshifted('.'),
	key.CodeKeypadPlusSign:    unshifted('+'),
	key.CodeKeypadHyphenMinus: unshifted('-'),
	key.CodeKeypadSlash:       unshifted('/'),

	key.Code0: shifted('0', ')'),
	key.Code1: shifted('1', '!'),
	key.Code2: shifted('2', '@'),
	key.Code3: shifted('3', '#'),
	key.Code4: shifted('4', '$'),
	key.Code5: shifted('5', '%'),
	key.Code6: shifted('6', '^'),
	key.Code7: shifted('7', '&'),
	key.Code8: shifted('8', '*'),
	key.Code9: shifted('9', '('),

	key.CodeA: uppercase('a'),
	key.CodeB: uppercase('b'),
	key.CodeC: uppercase('c'),
	key.CodeD: uppercase('d'),
	key.CodeE: uppercase('e'),
	key.CodeF: uppercase('f'),
	key.CodeG: uppercase('g'),
	key.CodeH: uppercase('h'),
	key.CodeI: uppercase('i'),
	key.CodeJ: uppercase('j'),
	key.CodeK: uppercase('k'),
	key.CodeL: uppercase('l'),
	key.CodeM: uppercase('m'),
	key.CodeN: uppercase('n'),
	key.CodeO: uppercase('o'),
	key.CodeP: uppercase('p'),
	key.CodeQ: uppercase('q'),
	key.CodeR: uppercase('r'),
	key.CodeS: uppercase('s'),
	key.CodeT: uppercase('t'),
	key.CodeU: uppercase('u'),
	key.CodeV: uppercase('v'),
	key.CodeW: uppercase('w'),
	key.CodeX: uppercase('x'),
	key.CodeY: uppercase('y'),
	key.CodeZ: uppercase('z'),

	key.CodeHyphenMinus:        shifted('-', '_'),
	key.CodeEqualSign:          shifted('=', '+'),
	key.CodeLeftSquareBracket:  shifted('[', '{'),
	key.CodeRightSquareBracket: shifted(']', '}'),
	key.CodeBackslash:          shifted('\\', '|'),
	key.CodeSemicolon:          shifted(';', ':'),
	key.CodeApostrophe:         shifted('\'', '"'),
	key.CodeGraveAccent:        shifted('`', '~'),
	key.CodeComma:              shifted(',', '<'),
	key.CodeFullStop:           shifted('.', '>'),
	key.CodeSlash:              shifted('/', '?'),
	key.CodeSpacebar:           unshifted(' '),
}
