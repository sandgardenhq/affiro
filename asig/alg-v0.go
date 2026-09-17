package asig

import (
	"time"
)

func (l *Asig) writeV0(ev Event) {
	now := time.Now().Unix()
	nowIndex := now >> 2
	currentIndex := l.CurrentSecond >> 2
	// Note: its not important that events get appended to the precise right byte, the algorithm does not rely
	// on each individual byte representing absolutely what occurred during a 4 second window, and
	// is generally tolerant of significant lag.
	// Hence, we are not tracking event time pre-lock (if we did, we could not confidently assume 'now' is always going
	// up)
	newBytes := nowIndex - currentIndex
	if newBytes > 0 {
		zeroBytes := make([]byte, newBytes)
		l.Data = append(l.Data, zeroBytes...) // Note: 0 represents a byte wherein no events occurred
		l.CurrentSecond = now
	}
	// If nowIndex == currentIndex, this is an existing byte with ~some data already
	last := len(l.Data) - 1
	l.Data[last] = applyEvent(l.Data[last], ev)
}

// Version 0:
// What events do we care about? events which can represent a document changing by a regular user
// so:
// keyup (we might care about the key that was pressed)
// mouseup (we do not care about position)
// We also specifically care about CTRL/CMD+V
// - paste
// We'll just increment by 1 for all other events; set a reserved bit
// The fastest typists in the world type 1500 characters per minute
// (1500 / 60 * 4) = 100, so we reserve 0-127 to note these events
// we could probably just reserve 0-63 for realistic cases.
// 1 typing / mouseup
// 2 typing / mouseup
// 4 typing / mouseup
// 8 typing / mouseup
// 16 typing / mouseup
// 32 typing / mouseup
// 64 typing / mouseup
// 128 - paste bit

// Q: what happens when models figure out how to mimick this algorithm?
// A: we bump the version and change it; moving target defense (tm)
