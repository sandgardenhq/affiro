package asig

import (
	"bytes"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"golang.org/x/mobile/event/key"
)

func TestNew(t *testing.T) {
	t.Parallel()
	t.Run("V0", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) {
			sig := New()
			if sig.Version != LatestStableVersion {
				t.Errorf("expected to get the latest signature version, was: %v", sig.Version)
			}
			sig.Version = Version0
			err := sig.Write(KeyDownEvent{})
			if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
			err = sig.Write(MouseUpEvent{})
			if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
			time.Sleep(100 * time.Second)
			err = sig.Write(KeyDownEvent{
				Key:            key.CodeV,
				ControlPressed: true,
			})
			if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
			expectedSig := "0.gENtOAAAAAA..AwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAIA"
			if sig.String() != expectedSig {
				t.Errorf("expected signature %v, got %v", expectedSig, sig.String())
			}
			sig.CurrentSecond = 0
			err = sig.Write(KeyDownEvent{})
			if !errors.Is(err, ErrUnwritableSignature) {
				t.Errorf("expected unwritable signature error, got: %v", err)
			}
			sig.CurrentSecond = 1
			sig.Version = 1002
			err = sig.Write(KeyDownEvent{})
			if !errors.Is(err, ErrUnknownVersion) {
				t.Errorf("expected unknown version error, got: %v", err)
			}
			sig.MustWrite(MouseUpEvent{})
		})
	})
	t.Run("V1", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) {
			sig := New()
			sig.Version = Version1
			err := sig.Write(KeyDownEvent{})
			if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
			err = sig.Write(MouseUpEvent{})
			if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
			time.Sleep(100 * time.Second)
			err = sig.Write(KeyDownEvent{
				Key:            key.CodeV,
				ControlPressed: true,
			})
			if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
			time.Sleep(4 * time.Second)
			err = sig.Write(KeyDownEvent{})
			if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
			time.Sleep(4 * time.Second)
			err = sig.Write(KeyDownEvent{})
			if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
			expectedSig := "1.gENtOAAAAAA..AwAZgAEB"
			if sig.String() != expectedSig {
				t.Errorf("expected signature %v, got %v", expectedSig, sig.String())
			}
			data := sig.Data
			// 3 key presses, pause, 100 seconds of pause, paste bit
			expectedBytes := []byte{3, 0, 100 / 4, 128, 1, 1}
			if !bytes.Equal(data, expectedBytes) {
				t.Errorf("expected signature bytes %v got %v", expectedBytes, data)
			}
		})
	})
}
