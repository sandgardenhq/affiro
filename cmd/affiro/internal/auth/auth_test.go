package auth

import "testing"

func TestAuthenticateLong(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	tok, err := Authenticate(t.Context(), "http://127.0.0.1:10380")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	t.Log(tok)
}
