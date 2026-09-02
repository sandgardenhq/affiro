package state

import (
	"context"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/sandgardenhq/affiro/asig"
	"github.com/sandgardenhq/affiro/cmd/affiro/internal/asigx"
)

type State struct {
	mu         sync.Mutex
	dir        string
	sig        *asigx.Asig
	cur        *os.File
	AuthToken  string
	AuthedAs   string
	RemoteHost string

	idleTimeout   time.Duration
	resetDuration time.Duration
	ResetAt       time.Time
	lastEventAt   time.Time
	StartTime     time.Time
}

func New(ctx context.Context, dir string, resetDuration time.Duration, host string) (*State, error) {
	s := &State{
		dir:           dir,
		RemoteHost:    host,
		resetDuration: resetDuration,
		idleTimeout:   5 * time.Minute,
		lastEventAt:   time.Now(),
	}
	sig := asigx.New()
	if err := s.Reset(sig); err != nil {
		return nil, err
	}
	go func() {
		t := time.NewTicker(resetDuration)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				sig := asigx.New()
				err := s.Reset(sig)
				if err != nil {
					fmt.Println("failed to reset signature tracking:", err)
				}
			}
		}
	}()
	return s, nil
}

func (s *State) Reset(newSig *asigx.Asig) error {
	if err := s.Refresh(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cur != nil {
		if err := s.cur.Close(); err != nil {
			return fmt.Errorf("failed to close old signature file: %w", err)
		}
	}
	s.sig = newSig
	filename := time.Unix(newSig.StartSecond, 0).Format("20060102_150405")
	filename = filepath.Join(s.dir, filename)
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create new signature file: %w", err)
	}
	s.cur = f
	s.StartTime = time.Now()
	s.ResetAt = s.StartTime.Add(s.resetDuration)
	return nil
}

func (s *State) Refresh() error {
	if s.sig == nil {
		return nil
	}
	content := s.sig.String()
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.cur.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if _, err := s.cur.Write([]byte(content)); err != nil {
		return err
	}
	return nil
}

func (s *State) ListRecent(limit int) ([]*asigx.Asig, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	out := make([]*asigx.Asig, 0, len(entries))
	for _, entry := range entries {
		f, err := os.ReadFile(filepath.Join(s.dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		ls, err := asigx.ParseString(string(f))
		if err != nil {
			continue
		}
		out = append(out, ls)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].StartSecond > out[j].StartSecond
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *State) String() string {
	return s.sig.Asig.String()
}

func (s *State) QRCode() (image.Image, error) {
	return s.sig.QRCode()
}

func (s *State) Write(ev asig.Event) error {
	s.mu.Lock()
	if time.Since(s.lastEventAt) > s.idleTimeout {
		// Note this will omit the wait period at the end of the signature; this should be OK;
		// a terminal wait period should in theory do nothing for analysis purposes
		sig := asigx.New()
		s.mu.Unlock()
		if err := s.Reset(sig); err != nil {
			fmt.Println("error resetting after idle timeout:", err)
		}
		s.mu.Lock()
	}
	s.lastEventAt = time.Now()
	s.mu.Unlock()
	return s.sig.Write(ev)
}

func (s *State) MustWrite(ev asig.Event) {
	_ = s.Write(ev)
}
