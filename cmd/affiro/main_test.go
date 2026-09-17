package main

import (
	"testing"

	"github.com/sandgardenhq/affiro/client"
)

//nolint:paralleltest
func TestAPIBaseURL(t *testing.T) {
	t.Run("defaults to prod when AFFIRO_API_BASE_URL is unset", func(t *testing.T) {
		t.Setenv("AFFIRO_API_BASE_URL", "")
		// Spelled out rather than compared to client.DefaultHost, which apiBaseURL returns:
		// that comparison would hold however the constant changed.
		const prod = "https://app.affiro.com"
		got := apiBaseURL()
		if got != prod {
			t.Errorf("apiBaseURL() = %q, want %q", got, prod)
		}
		if client.DefaultHost != prod {
			t.Errorf("client.DefaultHost = %q, want %q", client.DefaultHost, prod)
		}
	})

	t.Run("uses AFFIRO_API_BASE_URL when set", func(t *testing.T) {
		t.Setenv("AFFIRO_API_BASE_URL", "http://127.0.0.1:10380")
		got := apiBaseURL()
		if got != "http://127.0.0.1:10380" {
			t.Errorf("apiBaseURL() = %q, want %q", got, "http://127.0.0.1:10380")
		}
	})
}
