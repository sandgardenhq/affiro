package asig

import (
	"bytes"
	"errors"
	"math/rand/v2"
	"testing"

	"golang.org/x/mobile/event/key"
)

func TestErrorTypes(t *testing.T) {
	t.Parallel()
	base := errors.New("err")
	errs := []error{
		BadVersionError{Err: base},
		BadStartError{Err: base},
		BadDataError{Err: base},
	}
	for _, err := range errs {
		if err.Error() == "" {
			t.Errorf("expected populated error string")
		}
		if !errors.Is(err, base) {
			t.Errorf("expected child error to be maintained")
		}
	}
}

func TestParseString(t *testing.T) {
	t.Parallel()
	type testCase struct {
		str    string
		expect *Asig
		check  func(t testing.TB, err error)
	}
	tcs := []testCase{
		{
			str: "not.enough.separators",
			check: func(t testing.TB, err error) {
				if !errors.Is(err, ErrInsufficientDividerBytes) {
					t.Fatalf("expected insufficient divider bytes error, got: %v", err)
				}
			},
		},
		{
			str: "string....",
			check: func(t testing.TB, err error) {
				if _, ok := errors.AsType[BadVersionError](err); !ok {
					t.Fatalf("expected bad version error, got: %v", err)
				}
			},
		},
		{
			str: "0.string...",
			check: func(t testing.TB, err error) {
				if _, ok := errors.AsType[BadStartError](err); !ok {
					t.Fatalf("expected bad start error, got: %v", err)
				}
			},
		},
		{
			str: "0.1..",
			check: func(t testing.TB, err error) {
				if _, ok := errors.AsType[BadStartError](err); !ok {
					t.Fatalf("expected bad start error, got: %v", err)
				}
			},
		},
		{
			str: "0.CHBGagAAAAA..",
			expect: &Asig{
				Version:     0,
				StartSecond: 1783001096,
				Data:        []byte{0},
			},
			check: func(t testing.TB, err error) {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
			},
		},
		{
			str: "0.CHBGagAAAAA..notbase64",
			check: func(t testing.TB, err error) {
				if _, ok := errors.AsType[BadDataError](err); !ok {
					t.Fatalf("expected bad data error, got: %v", err)
				}
			},
		},
		{
			str: "0.CHBGagAAAAA..aaabbb",
			expect: &Asig{
				Version:     0,
				StartSecond: 1783001096,
				Data:        []byte{105, 166, 155, 109},
			},
			check: func(t testing.TB, err error) {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.str, func(t *testing.T) {
			t.Parallel()
			got, err := ParseString(tc.str)
			asigEqual(t, got, tc.expect)
			tc.check(t, err)
		})
	}
}

func TestStringEncoding(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name string
		sig  *Asig
	}
	tcs := []testCase{
		{
			name: "empty",
			sig: &Asig{
				Data: make([]byte, 1),
			},
		}, {
			name: "mouse_a",
			sig: func() *Asig {
				a := New()
				a.MustWrite(MouseUpEvent{})
				a.MustWrite(KeyDownEvent{
					Key: key.CodeA,
				})
				return a
			}(),
		}, {
			name: "random",
			sig: func() *Asig {
				version := rand.IntN(200)
				startSecond := rand.Int64N(10000000)
				currentSecond := rand.Int64N(10000000)
				dataLen := rand.IntN(100) + 1
				data := make([]byte, dataLen)
				for i := range data {
					data[i] = byte(rand.IntN(255))
				}
				return &Asig{
					Version:       Version(version),
					StartSecond:   startSecond,
					CurrentSecond: currentSecond,
					Data:          data,
				}
			}(),
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := tc.sig.String()
			t.Log(s)
			parsed, err := ParseString(s)
			if err != nil {
				t.Fatalf("failed to reparse encoded string: %v", err)
			}
			parsed.CurrentSecond = tc.sig.CurrentSecond
			asigEqual(t, parsed, tc.sig)
		})
	}
}

func asigEqual(t testing.TB, a, b *Asig) {
	if b == nil {
		if a != nil {
			t.Fatalf("expected nil, got non-nil")
		}
		return
	}
	if a == nil {
		t.Fatalf("expected non-nil, got nil")
		return
	}
	if a.CurrentSecond != b.CurrentSecond {
		t.Fatalf("current second mismatch: got %v expected %v", a.CurrentSecond, b.CurrentSecond)
	}
	if a.StartSecond != b.StartSecond {
		t.Fatalf("start second mismatch: got %v expected %v", a.StartSecond, b.StartSecond)
	}
	if a.Version != b.Version {
		t.Fatalf("version mismatch: got %v expected %v", a.Version, b.Version)
	}
	if !bytes.Equal(a.Data, b.Data) {
		t.Log(a.Data)
		t.Log(b.Data)
		t.Fatalf("data mismatch")
	}
}
