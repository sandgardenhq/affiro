//go:build darwin

package osxdriver

import (
	"fmt"

	"golang.org/x/mobile/event/key"
)

type KeyCode uint16

const (
	KeyCodeNoEvent         KeyCode = 32000
	KeyCodeA               KeyCode = 0
	KeyCodeS               KeyCode = 1
	KeyCodeD               KeyCode = 2
	KeyCodeF               KeyCode = 3
	KeyCodeH               KeyCode = 4
	KeyCodeG               KeyCode = 5
	KeyCodeZ               KeyCode = 6
	KeyCodeX               KeyCode = 7
	KeyCodeC               KeyCode = 8
	KeyCodeV               KeyCode = 9
	KeyCodeB               KeyCode = 11
	KeyCodeQ               KeyCode = 12
	KeyCodeW               KeyCode = 13
	KeyCodeE               KeyCode = 14
	KeyCodeR               KeyCode = 15
	KeyCodeY               KeyCode = 16
	KeyCodeT               KeyCode = 17
	KeyCode1               KeyCode = 18
	KeyCode2               KeyCode = 19
	KeyCode3               KeyCode = 20
	KeyCode4               KeyCode = 21
	KeyCode6               KeyCode = 22
	KeyCode5               KeyCode = 23
	KeyCodeEquals          KeyCode = 24
	KeyCode9               KeyCode = 25
	KeyCode7               KeyCode = 26
	KeyCodeMinus           KeyCode = 27
	KeyCode8               KeyCode = 28
	KeyCode0               KeyCode = 29
	KeyCodeRightBracket    KeyCode = 30
	KeyCodeO               KeyCode = 31
	KeyCodeU               KeyCode = 32
	KeyCodeLeftBracket     KeyCode = 33
	KeyCodeI               KeyCode = 34
	KeyCodeP               KeyCode = 35
	KeyCodeReturn          KeyCode = 36
	KeyCodeL               KeyCode = 37
	KeyCodeJ               KeyCode = 38
	KeyCodeApostraphe      KeyCode = 39
	KeyCodeK               KeyCode = 40
	KeyCoeSemicolon        KeyCode = 41
	KeyCodeBackSlash       KeyCode = 42
	KeyCodeComma           KeyCode = 43
	KyeCodeForwardSlash    KeyCode = 44
	KeyCodeN               KeyCode = 45
	KeyCodeM               KeyCode = 46
	KeyCodePeriod          KeyCode = 47
	KeyCodeTab             KeyCode = 48
	KeyCodeSpacebar        KeyCode = 49
	KeyCodeTilde           KeyCode = 50
	KeyCodeBackspace       KeyCode = 51
	KeyCodeEscape          KeyCode = 53
	KeyCodeCommand         KeyCode = 55
	KeyCodeShift           KeyCode = 56
	KeyCodeCapsLock        KeyCode = 57
	KeyCodeOption          KeyCode = 58
	KeyCodeControl         KeyCode = 59
	KeyCodeNumperiod       KeyCode = 65
	KeyCodeAsterisk        KeyCode = 67
	KeyCodeNumPlus         KeyCode = 69
	KeyCodeNumLock         KeyCode = 71
	KeyCodeNumForwardSlash KeyCode = 75
	KeyCodeNumEnter        KeyCode = 76
	KeyCodeNum0            KeyCode = 82
	KeyCodeNum1            KeyCode = 83
	KeyCodeNum2            KeyCode = 84
	KeyCodeNum3            KeyCode = 85
	KeyCodeNum4            KeyCode = 86
	KeyCodeNum5            KeyCode = 87
	KeyCodeNum6            KeyCode = 88
	KeyCodeNum7            KeyCode = 89
	KeyCodeNum8            KeyCode = 91
	KeyCodeNum9            KeyCode = 92
	KeyCodeF3              KeyCode = 99
	KeyCodeF5              KeyCode = 96
	KeyCodeF6              KeyCode = 97
	KeyCodeF7              KeyCode = 98
	KeyCodeF8              KeyCode = 100
	KeyCodeF9              KeyCode = 101
	KeyCodeF11             KeyCode = 103
	KeyCodeF13             KeyCode = 105
	KeyCodeF14             KeyCode = 107
	KeyCodeF10             KeyCode = 109
	KeyCodeF12             KeyCode = 111
	KeyCodeF15             KeyCode = 113
	KeyCodeInsert          KeyCode = 114
	KeyCodeHome            KeyCode = 115
	KeyDownPageUp          KeyCode = 116
	KeyCodeDelete          KeyCode = 117
	KeyCodeF4              KeyCode = 118
	KeyCodeEnd             KeyCode = 119
	KeyCodeF2              KeyCode = 120
	KeyCodePageDown        KeyCode = 121
	KeyCodeF1              KeyCode = 122
	KeyCodeLeftArrow       KeyCode = 123
	KeyCodeRightArrow      KeyCode = 124
	KeyCodeDownArrow       KeyCode = 125
	KeyCodeUpArrow         KeyCode = 126
)

var keycodeMapping = map[KeyCode]key.Code{
	KeyCodeV:               key.CodeV,
	KeyCode0:               key.Code0,
	KeyCode1:               key.Code1,
	KeyCode2:               key.Code2,
	KeyCode3:               key.Code3,
	KeyCode4:               key.Code4,
	KeyCode5:               key.Code5,
	KeyCode6:               key.Code6,
	KeyCode7:               key.Code7,
	KeyCode8:               key.Code8,
	KeyCode9:               key.Code9,
	KeyCodeA:               key.CodeA,
	KeyCodeB:               key.CodeB,
	KeyCodeC:               key.CodeC,
	KeyCodeD:               key.CodeD,
	KeyCodeE:               key.CodeE,
	KeyCodeF:               key.CodeF,
	KeyCodeG:               key.CodeG,
	KeyCodeH:               key.CodeH,
	KeyCodeI:               key.CodeI,
	KeyCodeJ:               key.CodeJ,
	KeyCodeK:               key.CodeK,
	KeyCodeL:               key.CodeL,
	KeyCodeM:               key.CodeM,
	KeyCodeN:               key.CodeN,
	KeyCodeO:               key.CodeO,
	KeyCodeP:               key.CodeP,
	KeyCodeQ:               key.CodeQ,
	KeyCodeR:               key.CodeR,
	KeyCodeS:               key.CodeS,
	KeyCodeT:               key.CodeT,
	KeyCodeU:               key.CodeU,
	KeyCodeW:               key.CodeW,
	KeyCodeX:               key.CodeX,
	KeyCodeY:               key.CodeY,
	KeyCodeZ:               key.CodeZ,
	KeyCodeApostraphe:      key.CodeApostrophe,
	KeyCodeAsterisk:        key.CodeKeypadAsterisk,
	KeyCodeBackSlash:       key.CodeBackslash,
	KeyCodeBackspace:       key.CodeDeleteBackspace,
	KeyCodeCapsLock:        key.CodeCapsLock,
	KeyCodeComma:           key.CodeComma,
	KeyCodeCommand:         key.CodeLeftGUI,
	KeyCodeControl:         key.CodeLeftControl,
	KeyCodeDelete:          key.CodeDeleteForward,
	KeyCodeEnd:             key.CodeEnd,
	KeyCodeEquals:          key.CodeEqualSign,
	KeyCodeEscape:          key.CodeEscape,
	KeyCodeF1:              key.CodeF1,
	KeyCodeF2:              key.CodeF2,
	KeyCodeF3:              key.CodeF3,
	KeyCodeF4:              key.CodeF4,
	KeyCodeF5:              key.CodeF5,
	KeyCodeF6:              key.CodeF6,
	KeyCodeF7:              key.CodeF7,
	KeyCodeF8:              key.CodeF8,
	KeyCodeF9:              key.CodeF9,
	KeyCodeF10:             key.CodeF10,
	KeyCodeF11:             key.CodeF11,
	KeyCodeF12:             key.CodeF12,
	KeyCodeF13:             key.CodeF13,
	KeyCodeF14:             key.CodeF14,
	KeyCodeF15:             key.CodeF15,
	KeyCodeHome:            key.CodeHome,
	KeyCodeInsert:          key.CodeInsert,
	KeyCodeLeftBracket:     key.CodeLeftSquareBracket,
	KeyCodeMinus:           key.CodeHyphenMinus,
	KeyCodeNum0:            key.CodeKeypad0,
	KeyCodeNum1:            key.CodeKeypad1,
	KeyCodeNum2:            key.CodeKeypad2,
	KeyCodeNum3:            key.CodeKeypad3,
	KeyCodeNum4:            key.CodeKeypad4,
	KeyCodeNum5:            key.CodeKeypad5,
	KeyCodeNum6:            key.CodeKeypad6,
	KeyCodeNum7:            key.CodeKeypad7,
	KeyCodeNum8:            key.CodeKeypad8,
	KeyCodeNum9:            key.CodeKeypad9,
	KeyCodeNumEnter:        key.CodeKeypadEnter,
	KeyCodeNumForwardSlash: key.CodeKeypadSlash,
	KeyCodeNumLock:         key.CodeKeypadNumLock,
	KeyCodeNumPlus:         key.CodeKeypadPlusSign,
	KeyCodeNumperiod:       key.CodeKeypadFullStop,
	KeyCodeOption:          key.CodeLeftAlt,
	KeyCodePageDown:        key.CodePageDown,
	KeyCodePeriod:          key.CodeFullStop,
	KeyCodeReturn:          key.CodeReturnEnter,
	KeyCodeRightBracket:    key.CodeRightSquareBracket,
	KeyCodeShift:           key.CodeLeftShift,
	KeyCodeSpacebar:        key.CodeSpacebar,
	KeyCodeTab:             key.CodeTab,
	KeyCodeTilde:           key.CodeGraveAccent,
	KeyCoeSemicolon:        key.CodeSemicolon,
	KeyDownPageUp:          key.CodePageUp,
	KyeCodeForwardSlash:    key.CodeSlash,
	KeyCodeDownArrow:       key.CodeDownArrow,
	KeyCodeLeftArrow:       key.CodeLeftArrow,
	KeyCodeRightArrow:      key.CodeRightArrow,
	KeyCodeUpArrow:         key.CodeUpArrow,
}

func convertKeyCode(kc KeyCode) key.Code {
	// TODO: osx key codes do not represent if the key
	// pressed is a shifted key e.g. parens for shift + 9/0;
	// we could encode that here, though, with modifiers.
	kc2, ok := keycodeMapping[kc]
	if ok {
		return kc2
	}
	fmt.Println("unknown key code", kc)
	return 0
}
