package asig

import (
	"strings"
)

// adapted from https://github.com/barklan/go-cryptobin/blob/v1.0.1/rand/randomart/randomart.go#L142

type RandomartSize uint8

const (
	RandomartSizeSmall RandomartSize = iota
	RandomartSizeMid
	RandomartSizeLarge
)

func sizeToDimensions(sz RandomartSize) (int, int) {
	var sizeY, sizeX = 9, 17
	switch sz {
	case RandomartSizeSmall:
		sizeY = 5
		sizeX = 9
	case RandomartSizeMid:
		sizeY = 7
		sizeX = 13
	case RandomartSizeLarge:
		// default
	}
	return sizeY, sizeX
}

func (a *Asig) Randomart(sz RandomartSize) string {
	sizeY, sizeX := sizeToDimensions(sz)
	charSet := " .o-=/\\|^v<>OLS"
	field := make([][]int, sizeX)
	for i := range field {
		field[i] = make([]int, sizeY)
	}
	maxChar := len(charSet) - 1

	x, y := sizeX/2, sizeY/2

	for _, input := range a.Data {
		for range 4 {
			if input&0x1 > 0 {
				x += 1
			} else {
				x += -1
			}

			if input&0x2 > 0 {
				y++
			} else {
				y--
			}

			x = min(max(x, 0), sizeX-1)
			y = min(max(y, 0), sizeY-1)

			if field[x][y] < maxChar-2 {
				field[x][y]++
			}
			input >>= 2
		}
	}

	field[sizeX/2][sizeY/2] = maxChar - 1
	field[x][y] = maxChar

	sb := strings.Builder{}
	sb.WriteRune('╭')
	sb.WriteString(strings.Repeat("─", sizeX))
	sb.WriteRune('╮')
	sb.WriteByte('\n')

	for y := range sizeY {
		sb.WriteRune('│')
		for x := range sizeX {
			sb.WriteByte(charSet[field[x][y]])
		}
		sb.WriteRune('│')
		sb.WriteByte('\n')
	}

	sb.WriteRune('╰')
	sb.WriteString(strings.Repeat("─", sizeX))
	sb.WriteRune('╯')

	return sb.String()
}

// q: why no ParseRandomart?
// a: randomart does not encode its directionality, so for that reason at least we can't reconstruct a byte sequence from it
