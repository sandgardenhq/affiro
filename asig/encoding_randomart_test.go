package asig

import (
	"math/rand/v2"
	"testing"

	"golang.org/x/mobile/event/key"
)

func TestRandomartEncoding(t *testing.T) {
	t.Parallel()
	type testCase struct {
		sig  *Asig
		name string
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
			s := tc.sig.Randomart(RandomartSizeSmall)
			t.Log(s)
			s = tc.sig.Randomart(RandomartSizeMid)
			t.Log(s)
			s = tc.sig.Randomart(RandomartSizeLarge)
			t.Log(s)
		})
	}
}
