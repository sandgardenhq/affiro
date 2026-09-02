package asig

import (
	"encoding/binary"
	"errors"
)

func (l *Asig) Bytes() []byte {
	buff := []byte{}
	buff = binary.LittleEndian.AppendUint64(buff, uint64(l.Version))
	buff = binary.LittleEndian.AppendUint64(buff, uint64(l.StartSecond))
	// 4 placeholder bytes for extra information in the future
	buff = append(buff, []byte{0, 0, 0, 0}...)
	// depending on the value of the 4 placeholder bytes, we may parse more fields here in the future
	buff = append(buff, l.Data...)
	return buff
}

var ErrBytesTooShort = errors.New("insufficient data to parse signature from bytes")

// ParseBytes parses a binary asig produced by the Bytes method; the resulting structure
// can be used in verification but cannot be written to without producing an invalid
// signature.
func ParseBytes(b []byte) (*Asig, error) {
	if len(b) < 20 {
		return nil, ErrBytesTooShort
	}
	i := 0
	version := binary.LittleEndian.Uint64(b[i:])
	i += 8
	startSecond := binary.LittleEndian.Uint64(b[i:])
	i += 8
	i += 4
	data := b[i:]
	// data must be at least one byte
	if len(data) == 0 {
		data = append(data, 0)
	}
	return &Asig{
		Version:     Version(version),
		StartSecond: int64(startSecond),
		Data:        data,
	}, nil
}
