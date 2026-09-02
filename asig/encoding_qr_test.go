package asig

import (
	"image/png"
	"math/rand/v2"
	"os"
	"testing"
)

func TestQREncoding(t *testing.T) {
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
	testFileOutputs := true
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			img, err := tc.sig.QRCode()
			if err != nil {
				t.Fatalf("failed to encode qr code: %v", err)
			}
			if testFileOutputs {
				f, err := os.Create(tc.name + "_test.png")
				if err != nil {
					t.Fatalf("failed to create png: %v", err)
				}
				defer func() {
					err = f.Close()
					if err != nil {
						t.Fatalf("failed to close created file: %v", err)
					}
				}()
				err = png.Encode(f, img)
				if err != nil {
					t.Fatalf("failed to encode png: %v", err)
				}
			}
			// BUG(200sc): this fails about 1/100 times
			parsed, err := ParseQRCode(img)
			if err != nil {
				t.Fatalf("failed to re-parse qr code: %v", err)
			}
			parsed.CurrentSecond = tc.sig.CurrentSecond
			asigEqual(t, tc.sig, parsed)
		})
	}
}
