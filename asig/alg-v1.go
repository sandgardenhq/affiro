package asig

import (
	"time"
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
	// If nowIndex == currentIndex, this is an existing byte with ~some data already
	last := len(l.Data) - 1
	l.Data[last] = applyEvent(l.Data[last], ev)
}

// Version 1:
// v0, but pauses are now two byes; 0 then the length of the pause / 4 seconds
