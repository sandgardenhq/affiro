package asig

import (
	"errors"
	"sync"
	"time"
)

type Version uint64

const (
	Version0            Version = 0
	Version1            Version = 1
	LatestStableVersion Version = Version1
)

type Asig struct {
	mu            sync.Mutex
	Data          []byte // 1 byte per 5 seconds
	Version       Version
	StartSecond   int64
	CurrentSecond int64
}

func New() *Asig {
	start := time.Now().Unix()
	return NewCustom(CustomOptions{
		Version:     LatestStableVersion,
		StartSecond: start,
	})
}

type CustomOptions struct {
	Version     Version
	StartSecond int64
}

func NewCustom(opts CustomOptions) *Asig {
	return &Asig{
		Version:       opts.Version,
		StartSecond:   opts.StartSecond,
		CurrentSecond: opts.StartSecond,
		Data:          make([]byte, 1),
	}
}

var ErrUnwritableSignature = errors.New("this signature is not writable; it was likely parsed instead of constructed in memory")
var ErrUnknownVersion = errors.New("invalid signature version")

func (l *Asig) MustWrite(ev Event) {
	_ = l.Write(ev)
}

func (l *Asig) Write(ev Event) error {
	if l.CurrentSecond == 0 {
		return ErrUnwritableSignature
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	switch l.Version {
	default:
		return ErrUnknownVersion
	case Version1:
		l.writeV1(ev)
	case Version0:
		l.writeV0(ev)
	}
	return nil
}
