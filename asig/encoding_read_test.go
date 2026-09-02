package asig

import (
	"io"
	"strings"
	"testing"
)

func TestRead(t *testing.T) {
	t.Parallel()
	t.Run("NoNewline", func(t *testing.T) {
		t.Parallel()
		_, _, err := Read(strings.NewReader("nonewline"))
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})
	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		sig, leftover, err := Read(strings.NewReader("0.CHBGagAAAAA..aaabbb\ncontent"))
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		content, err := io.ReadAll(leftover)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if string(content) != "content" {
			t.Errorf("expected to have 'content' left over, had: %q", string(content))
		}
		expect := &Asig{
			Version:     0,
			StartSecond: 1783001096,
			Data:        []byte{105, 166, 155, 109},
		}
		asigEqual(t, expect, sig)
	})
}
