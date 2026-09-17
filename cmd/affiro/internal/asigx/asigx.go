package asigx

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/sandgardenhq/affiro/asig"
)

type Asig struct {
	*asig.Asig
	FirstCharacters string
	TotalActions    int
	CharLimit       int
	mu              sync.Mutex
}

func New() *Asig {
	as := asig.New()
	return &Asig{
		Asig:      as,
		CharLimit: 15,
	}
}

func (a *Asig) MustWrite(ev asig.Event) {
	_ = a.Write(ev)
}

func (a *Asig) Write(ev asig.Event) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.FirstCharacters) < a.CharLimit {
		if v, ok := ev.(asig.KeyDownEvent); ok {
			// TODO: we probably shouldn't be emitting x00 strings
			if v.String != "" && v.String != "\x00" {
				a.FirstCharacters += v.String
			}
		}
	}
	a.TotalActions++
	if err := a.Asig.Write(ev); err != nil {
		return fmt.Errorf("writing the event to the signature: %w", err)
	}
	return nil
}

func (a *Asig) String() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	lines := []string{
		a.Asig.String(),
		a.FirstCharacters,
		strconv.Itoa(a.TotalActions),
	}
	return strings.Join(lines, "\n")
}

// ErrEmptyString is returned when there is nothing at all to parse a signature from.
var ErrEmptyString = errors.New("empty string")

func ParseString(s string) (*Asig, error) {
	sSplit := strings.Split(s, "\n")
	if len(sSplit) == 0 {
		return nil, ErrEmptyString
	}
	as, err := asig.ParseString(sSplit[0])
	if err != nil {
		return nil, fmt.Errorf("parsing the signature line: %w", err)
	}
	if len(sSplit) == 1 {
		return &Asig{
			Asig: as,
		}, nil
	}
	chars := sSplit[1]
	if len(sSplit) == 2 {
		return &Asig{
			Asig:            as,
			FirstCharacters: chars,
		}, nil
	}
	actions, err := strconv.Atoi(sSplit[2])
	if err != nil {
		return nil, fmt.Errorf("parsing the total action count: %w", err)
	}
	return &Asig{
		Asig:            as,
		FirstCharacters: chars,
		TotalActions:    actions,
	}, nil
}
