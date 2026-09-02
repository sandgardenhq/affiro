package asig

import (
	"time"

	"golang.org/x/mobile/event/key"
)

func (l *Asig) writeV1(ev Event) {
	now := time.Now().Unix()
	nowIndex := now >> 2
	currentIndex := l.CurrentSecond >> 2
	pauseLength := nowIndex - currentIndex
	if pauseLength > 0 {
		l.Data = append(l.Data, 0)
		if pauseLength > 1 {
			// encode length of pause
			l.Data = append(l.Data, byte(min(pauseLength, 255)))
			l.Data = append(l.Data, 0) // 0, to be incremented below
		}
		l.CurrentSecond = now
	}
	b := l.Data[len(l.Data)-1]
	switch v := ev.(type) {
	case KeyDownEvent:
		if (v.SpecialPressed || v.ControlPressed) && v.Key == key.CodeV {
			b |= pastedBit
		} else if b != 127 && b != 255 {
			b += 1
		}
	case MouseUpEvent:
		if b != 127 && b != 126 && b != 255 && b != 254 {
			b += 2
		}
	}
	// If nowIndex == currentIndex, this will be an existing byte with ~some data already
	l.Data[len(l.Data)-1] = b
}

// Version 1:
// v0, but pauses are now two byes; 0 then the length of the pause / 4 seconds
