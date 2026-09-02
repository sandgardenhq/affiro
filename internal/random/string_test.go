package random

import (
	"testing"
)

func TestString(t *testing.T) {
	t.Parallel()
	s := String(10)
	if len(s) != 10 {
		t.Errorf("expected len 10, got %v", len(s))
	}
}

func TestUpperString(t *testing.T) {
	t.Parallel()
	s := UpperString(10)
	if len(s) != 10 {
		t.Errorf("expected len 10, got %v", len(s))
	}
}
