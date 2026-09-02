package asigx

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/sandgardenhq/affiro/asig"
)

type Asig struct {
	*asig.Asig

	mu              sync.Mutex
	FirstCharacters string
	TotalActions    int
	CharLimit       int
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
		switch v := ev.(type) {
		case asig.KeyDownEvent:
			// TODO: we probably shouldn't be emitting x00 strings
			if v.String != "" && v.String != "\x00" {
				a.FirstCharacters += v.String
			}
			// else {
			// 	fmt.Println("(debug) no string for input event", v)
			// }
		}
	}
	// else {
	// 	fmt.Println("at limit")
	// }
	a.TotalActions++
	return a.Asig.Write(ev)
}

func (a *Asig) String() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	lines := []string{}
	lines = append(lines, a.Asig.String())
	lines = append(lines, a.FirstCharacters)
	lines = append(lines, strconv.Itoa(a.TotalActions))
	return strings.Join(lines, "\n")
}

func ParseString(s string) (*Asig, error) {
	sSplit := strings.Split(s, "\n")
	if len(sSplit) == 0 {
		return nil, fmt.Errorf("empty string")
	}
	as, err := asig.ParseString(sSplit[0])
	if err != nil {
		return nil, err
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
		return nil, err
	}
	return &Asig{
		Asig:            as,
		FirstCharacters: chars,
		TotalActions:    actions,
	}, nil
}
