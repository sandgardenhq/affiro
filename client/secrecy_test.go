package client_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sandgardenhq/affiro/client"
)

const theToken = "crd-0000000000000001.sup3rs3cr3t"

// Each case hands back the client's own error untouched: these tests assert on what the
// error says and which sentinel it matches, so wrapping would hide the thing under test.
//
//nolint:wrapcheck // the unwrapped error is the subject of the assertion
func failureModes(t *testing.T) map[string]func(*testing.T) error {
	t.Helper()
	statusOnly := func(status int, body string) func(*testing.T) error {
		return func(t *testing.T) error {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
				_, _ = w.Write([]byte(body))
			}))
			defer srv.Close()
			cl, err := client.New(theToken, client.WithHost(srv.URL))
			if err != nil {
				t.Fatalf("unexpected error building client: %v", err)
			}
			_, err = cl.Upload(t.Context(), testSignature(t), strings.NewReader("hello"))
			return err
		}
	}
	return map[string]func(*testing.T) error{
		"rejected token": statusOnly(http.StatusUnauthorized, `{"error":"access denied"}`),
		"bad document":   statusOnly(http.StatusBadRequest, `{"error":"failed to read signature"}`),
		"server failing": statusOnly(http.StatusInternalServerError, "upstream exploded"),
		"junk response":  statusOnly(http.StatusOK, "this is not json"),
		"a server that echoes the key": func(t *testing.T) error {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"rejected ` + r.Header.Get("Authorization") + `"}`))
			}))
			defer srv.Close()
			cl, err := client.New(theToken, client.WithHost(srv.URL))
			if err != nil {
				t.Fatalf("unexpected error building client: %v", err)
			}
			_, err = cl.Upload(t.Context(), testSignature(t), strings.NewReader("hello"))
			return err
		},
		"nil signature": func(t *testing.T) error {
			cl, err := client.New(theToken, client.WithHost("http://127.0.0.1:1"))
			if err != nil {
				t.Fatalf("unexpected error building client: %v", err)
			}
			_, err = cl.Upload(t.Context(), nil, strings.NewReader("hello"))
			return err
		},
		"no document id": func(t *testing.T) error {
			cl, err := client.New(theToken, client.WithHost("http://127.0.0.1:1"))
			if err != nil {
				t.Fatalf("unexpected error building client: %v", err)
			}
			_, err = cl.UploadVersion(t.Context(), "", testSignature(t), strings.NewReader("hello"))
			return err
		},
		"unreachable host": func(t *testing.T) error {
			srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
			host := srv.URL
			srv.Close()
			cl, err := client.New(theToken, client.WithHost(host))
			if err != nil {
				t.Fatalf("unexpected error building client: %v", err)
			}
			_, err = cl.Upload(t.Context(), testSignature(t), strings.NewReader("hello"))
			return err
		},
		"unusable host": func(t *testing.T) error {
			cl, err := client.New(theToken, client.WithHost("http://\x7f/"))
			if err != nil {
				return err
			}
			_, err = cl.Upload(t.Context(), testSignature(t), strings.NewReader("hello"))
			return err
		},
	}
}

func TestTokenNeverAppearsInAnError(t *testing.T) {
	t.Parallel()
	for name, failure := range failureModes(t) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			err := failure(t)
			if err == nil {
				t.Fatal("expected a failure, got none")
			}
			for rendering, text := range map[string]string{
				"Error()": err.Error(),
				"%v":      fmt.Sprintf("%v", err),
				"%+v":     fmt.Sprintf("%+v", err),
			} {
				if strings.Contains(text, theToken) {
					t.Errorf("%s leaks the API key: %s", rendering, text)
				}
			}
		})
	}
}
