package asig

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var stringSigDivider byte = '.'

func (l *Asig) String() string {
	sb := strings.Builder{}
	sb.WriteString(strconv.FormatInt(int64(l.Version), 10))
	sb.WriteByte(stringSigDivider)
	secBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(secBytes, uint64(l.StartSecond))
	sb.WriteString(base64.RawURLEncoding.EncodeToString(secBytes))
	sb.WriteByte(stringSigDivider)
	// placeholder for extra information
	sb.WriteByte(stringSigDivider)
	sb.WriteString(base64.RawURLEncoding.EncodeToString(l.Data))
	return sb.String()
}

var ErrInsufficientDividerBytes = errors.New("insufficient divider bytes in string signature")

type BadVersionError struct {
	Err error
}

func (e BadVersionError) Error() string {
	return fmt.Sprintf("bad version: %s", e.Err.Error())
}
func (e BadVersionError) Unwrap() error {
	return e.Err
}

type BadStartError struct {
	Err error
}

func (e BadStartError) Error() string {
	return fmt.Sprintf("bad start: %s", e.Err.Error())
}
func (e BadStartError) Unwrap() error {
	return e.Err
}

type BadDataError struct {
	Err error
}

func (e BadDataError) Error() string {
	return fmt.Sprintf("bad signature data: %s", e.Err.Error())
}
func (e BadDataError) Unwrap() error {
	return e.Err
}

// ParseString parses an asig produced by the String method; the resulting structure
// can be used in verification but cannot be written to without producing an invalid
// signature.
func ParseString(s string) (*Asig, error) {
	splitSig := strings.Split(s, string(stringSigDivider))
	if len(splitSig) < 4 {
		return nil, ErrInsufficientDividerBytes
	}
	version, err := strconv.ParseInt(splitSig[0], 10, 64)
	if err != nil {
		return nil, BadVersionError{Err: err}
	}
	secBytes, err := base64.RawURLEncoding.DecodeString(splitSig[1])
	if err != nil {
		return nil, BadStartError{Err: err}
	}
	if len(secBytes) != 8 {
		return nil, BadStartError{Err: errors.New("insufficient bytes")}
	}
	startSecond := int64(binary.LittleEndian.Uint64(secBytes))
	// splitSig-1, not 2, so extra information can be placed here
	data, err := base64.RawURLEncoding.DecodeString(splitSig[len(splitSig)-1])
	if err != nil {
		return nil, BadDataError{Err: err}
	}
	if len(data) == 0 {
		data = append(data, 0)
	}
	return &Asig{
		Data:        data,
		StartSecond: startSecond,
		Version:     Version(version),
	}, nil
}
