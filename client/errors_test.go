package client_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sandgardenhq/affiro/client"
)

func TestErrorKindByStatus(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		want   error
		status int
	}{
		"rejected token":     {status: http.StatusUnauthorized, want: client.ErrUnauthorized},
		"forbidden token":    {status: http.StatusForbidden, want: client.ErrUnauthorized},
		"malformed document": {status: http.StatusBadRequest, want: client.ErrInvalidDocument},
		"document too large": {status: http.StatusRequestEntityTooLarge, want: client.ErrInvalidDocument},
		"API failing":        {status: http.StatusInternalServerError, want: client.ErrUnavailable},
		"API down":           {status: http.StatusBadGateway, want: client.ErrUnavailable},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(`{"error":"nope"}`))
			}))
			defer srv.Close()

			cl, err := client.New("secret-token", client.WithHost(srv.URL))
			if err != nil {
				t.Fatalf("unexpected error building client: %v", err)
			}
			_, err = cl.Upload(t.Context(), testSignature(t), strings.NewReader("hello"))
			if !errors.Is(err, tc.want) {
				t.Fatalf("got error %v, want one matching %v", err, tc.want)
			}
			for _, other := range []error{client.ErrUnauthorized, client.ErrInvalidDocument, client.ErrUnavailable} {
				if !errors.Is(other, tc.want) && errors.Is(err, other) {
					t.Errorf("error also matches %v, so the caller cannot tell the three apart", other)
				}
			}
		})
	}
}

func TestUnreachableHostIsUnavailableNotUnauthorized(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	host := srv.URL
	srv.Close() // nothing is listening there now

	cl, err := client.New("secret-token", client.WithHost(host))
	if err != nil {
		t.Fatalf("unexpected error building client: %v", err)
	}
	_, err = cl.Upload(t.Context(), testSignature(t), strings.NewReader("hello"))
	if !errors.Is(err, client.ErrUnavailable) {
		t.Fatalf("got error %v, want one matching %v", err, client.ErrUnavailable)
	}
	if errors.Is(err, client.ErrUnauthorized) || errors.Is(err, client.ErrInvalidDocument) {
		t.Error("an unreachable host reads as a credential or document problem")
	}
}

func TestErrorCarriesStatusAndServerMessage(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"failed to read signature: unexpected EOF"}`))
	}))
	defer srv.Close()

	cl, err := client.New("secret-token", client.WithHost(srv.URL))
	if err != nil {
		t.Fatalf("unexpected error building client: %v", err)
	}
	_, err = cl.Upload(t.Context(), testSignature(t), strings.NewReader("hello"))

	var apiErr *client.Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("got error %v, want a *client.Error", err)
	}
	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("got status %d, want 400", apiErr.StatusCode)
	}
	if !strings.Contains(apiErr.Message, "failed to read signature") {
		t.Errorf("got message %q, want the API's own explanation", apiErr.Message)
	}
}
