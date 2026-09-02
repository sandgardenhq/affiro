package asig

import (
	"errors"
	"math/rand/v2"
	"testing"
)

func TestParseBytes(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name   string
		data   []byte
		expect *Asig
		check  func(t testing.TB, err error)
	}
	tcs := []testCase{
		{
			name: "tooShort",
			data: []byte{0, 0, 0, 0},
			check: func(t testing.TB, err error) {
				if !errors.Is(err, ErrBytesTooShort) {
					t.Fatalf("expected bytes too short error, got: %v", err)
				}
			},
		},
		{
			name: "zeroData",
			data: []byte{0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			expect: &Asig{
				Version:     0,
				StartSecond: 1,
				Data:        []byte{0},
			},
			check: func(t testing.TB, err error) {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
			},
		},
		{
			name: "someData",
			data: []byte{0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 3, 4, 5},
			expect: &Asig{
				Version:     0,
				StartSecond: 1,
				Data:        []byte{2, 3, 4, 5},
			},
			check: func(t testing.TB, err error) {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseBytes(tc.data)
			asigEqual(t, got, tc.expect)
			tc.check(t, err)
		})
	}
}

func TestBytesEncoding(t *testing.T) {
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
			s := tc.sig.Bytes()
			parsed, err := ParseBytes(s)
			if err != nil {
				t.Fatalf("failed to reparse encoded bytes: %v", err)
			}
			parsed.CurrentSecond = tc.sig.CurrentSecond
			asigEqual(t, parsed, tc.sig)
		})
	}
}
