package main

import "testing"

//nolint:paralleltest
func TestAPIBaseURL(t *testing.T) {
	t.Run("defaults to prod when AFFIRO_API_BASE_URL is unset", func(t *testing.T) {
		t.Setenv("AFFIRO_API_BASE_URL", "")
		got := apiBaseURL()
		if got != defaultAPIBaseURL {
			t.Errorf("apiBaseURL() = %q, want %q", got, defaultAPIBaseURL)
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
