package client_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sandgardenhq/affiro/client"
)

func TestDefaultHostIsUsedWhenNoneIsGiven(t *testing.T) {
	t.Parallel()
	var got string
	cl, err := client.New("secret-token", client.WithHTTPClient(&http.Client{
		Transport: transportRecordingURL(&got),
	}))
	if err != nil {
		t.Fatalf("unexpected error building client: %v", err)
	}
	_, _ = cl.Upload(t.Context(), testSignature(t), strings.NewReader("hello"))
	if want := client.DefaultHost + "/api/v1/documents"; got != want {
		t.Errorf("a client given no host posted to %q, want %q", got, want)
	}
}

func TestHostTrailingSlashDoesNotDoubleUp(t *testing.T) {
	t.Parallel()
	var got string
	cl, err := client.New("secret-token",
		client.WithHost("https://play.example.com/"),
		client.WithHTTPClient(&http.Client{Transport: transportRecordingURL(&got)}),
	)
	if err != nil {
		t.Fatalf("unexpected error building client: %v", err)
	}
	_, _ = cl.Upload(t.Context(), testSignature(t), strings.NewReader("hello"))
	if want := "https://play.example.com/api/v1/documents"; got != want {
		t.Errorf("posted to %q, want %q", got, want)
	}
}

func TestNewRejectsAnEmptyToken(t *testing.T) {
	t.Parallel()
	_, err := client.New("")
	if !errors.Is(err, client.ErrUnauthorized) {
		t.Errorf("got error %v, want one matching %v", err, client.ErrUnauthorized)
	}
}

func TestCancelledContextIsUnavailable(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer srv.Close()

	cl, err := client.New("secret-token", client.WithHost(srv.URL))
	if err != nil {
		t.Fatalf("unexpected error building client: %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err = cl.Upload(ctx, testSignature(t), strings.NewReader("hello"))
	if !errors.Is(err, context.Canceled) {
		t.Errorf("got error %v, want one matching context.Canceled", err)
	}
	if !errors.Is(err, client.ErrUnavailable) {
		t.Errorf("got error %v, want one matching %v", err, client.ErrUnavailable)
	}
}

// transportRecordingURL stands in for the API: it answers an empty 200 and records the URL it
// was asked for.
func transportRecordingURL(into *string) http.RoundTripper {
	return roundTripFunc(func(r *http.Request) (*http.Response, error) {
		*into = r.URL.String()
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       http.NoBody,
			Header:     http.Header{},
		}, nil
	})
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
