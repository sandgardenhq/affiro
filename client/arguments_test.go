package client_test

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sandgardenhq/affiro/client"
)

func refusingServer(t *testing.T) (srv *httptest.Server, reached *bool) {
	t.Helper()
	var wasReached bool
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		wasReached = true
		_, _ = io.WriteString(w, `{"id":"sdc-1"}`)
	}))
	t.Cleanup(srv.Close)
	return srv, &wasReached
}

func TestUploadRejectsANilSignature(t *testing.T) {
	t.Parallel()
	srv, reached := refusingServer(t)
	cl, err := client.New("secret-token", client.WithHost(srv.URL))
	if err != nil {
		t.Fatalf("unexpected error building client: %v", err)
	}

	_, err = cl.Upload(t.Context(), nil, strings.NewReader("hello"))
	if !errors.Is(err, client.ErrInvalidDocument) {
		t.Errorf("got error %v, want one matching %v", err, client.ErrInvalidDocument)
	}
	if *reached {
		t.Error("the client sent a request it should have refused to build")
	}
}

func TestUploadVersionRejectsANilSignature(t *testing.T) {
	t.Parallel()
	srv, reached := refusingServer(t)
	cl, err := client.New("secret-token", client.WithHost(srv.URL))
	if err != nil {
		t.Fatalf("unexpected error building client: %v", err)
	}

	_, err = cl.UploadVersion(t.Context(), "doc-1", nil, strings.NewReader("hello"))
	if !errors.Is(err, client.ErrInvalidDocument) {
		t.Errorf("got error %v, want one matching %v", err, client.ErrInvalidDocument)
	}
	if *reached {
		t.Error("the client sent a request it should have refused to build")
	}
}

func TestUploadRejectsNilContent(t *testing.T) {
	t.Parallel()
	srv, reached := refusingServer(t)
	cl, err := client.New("secret-token", client.WithHost(srv.URL))
	if err != nil {
		t.Fatalf("unexpected error building client: %v", err)
	}

	_, err = cl.Upload(t.Context(), testSignature(t), nil)
	if !errors.Is(err, client.ErrInvalidDocument) {
		t.Errorf("got error %v, want one matching %v", err, client.ErrInvalidDocument)
	}
	if *reached {
		t.Error("the client sent a request it should have refused to build")
	}
}

func TestUploadRejectsATypedNilContent(t *testing.T) {
	t.Parallel()
	var (
		nilBuffer *bytes.Buffer
		nilBytes  *bytes.Reader
		nilString *strings.Reader
	)
	for name, content := range map[string]io.Reader{
		"a nil bytes.Buffer":   nilBuffer,
		"a nil bytes.Reader":   nilBytes,
		"a nil strings.Reader": nilString,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			srv, reached := refusingServer(t)
			cl, err := client.New("secret-token", client.WithHost(srv.URL))
			if err != nil {
				t.Fatalf("unexpected error building client: %v", err)
			}

			_, err = cl.Upload(t.Context(), testSignature(t), content)
			if !errors.Is(err, client.ErrInvalidDocument) {
				t.Errorf("got error %v, want one matching %v", err, client.ErrInvalidDocument)
			}
			if *reached {
				t.Error("the client sent a request it should have refused to build")
			}
		})
	}
}

func TestUploadVersionRejectsAnUnusableDocumentID(t *testing.T) {
	t.Parallel()
	for name, documentID := range map[string]string{
		"empty":         "",
		"only spaces":   "   ",
		"a dot":         ".",
		"a parent walk": "..",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			srv, reached := refusingServer(t)
			cl, err := client.New("secret-token", client.WithHost(srv.URL))
			if err != nil {
				t.Fatalf("unexpected error building client: %v", err)
			}

			_, err = cl.UploadVersion(t.Context(), documentID, testSignature(t), strings.NewReader("hello"))
			if !errors.Is(err, client.ErrInvalidDocument) {
				t.Errorf("got error %v, want one matching %v", err, client.ErrInvalidDocument)
			}
			if *reached {
				t.Error("the client sent a request it should have refused to build")
			}
		})
	}
}

func TestWithHTTPClientIgnoresANilClient(t *testing.T) {
	t.Parallel()
	srv, _ := refusingServer(t)
	cl, err := client.New("secret-token", client.WithHost(srv.URL), client.WithHTTPClient(nil))
	if err != nil {
		t.Fatalf("unexpected error building client: %v", err)
	}
	if _, err := cl.Upload(t.Context(), testSignature(t), strings.NewReader("hello")); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewIgnoresANilOption(t *testing.T) {
	t.Parallel()
	if _, err := client.New("secret-token", nil); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewRejectsAHostItCannotReach(t *testing.T) {
	t.Parallel()
	for name, host := range map[string]string{
		"empty":       "",
		"only spaces": "  ",
		"no scheme":   "play.example.com",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := client.New("secret-token", client.WithHost(host))
			if err == nil {
				t.Fatalf("New accepted host %q", host)
			}
			if !errors.Is(err, client.ErrUnavailable) {
				t.Errorf("got error %v, want one matching %v", err, client.ErrUnavailable)
			}
		})
	}
}

//nolint:wrapcheck // each case returns the client's own error so the test can match it
func TestEveryErrorMatchesExactlyOneSentinel(t *testing.T) {
	t.Parallel()
	sentinels := []error{client.ErrUnauthorized, client.ErrInvalidDocument, client.ErrUnavailable}

	var nilBuffer *bytes.Buffer
	for name, produce := range map[string]func(*testing.T) error{
		"no API key": func(*testing.T) error {
			_, err := client.New("")
			return err
		},
		"unusable host": func(*testing.T) error {
			_, err := client.New("tok", client.WithHost(""))
			return err
		},
		"nil signature": func(t *testing.T) error {
			cl := mustClient(t)
			_, err := cl.Upload(t.Context(), nil, strings.NewReader("hello"))
			return err
		},
		"nil content": func(t *testing.T) error {
			cl := mustClient(t)
			_, err := cl.Upload(t.Context(), testSignature(t), nil)
			return err
		},
		"typed nil content": func(t *testing.T) error {
			cl := mustClient(t)
			_, err := cl.Upload(t.Context(), testSignature(t), nilBuffer)
			return err
		},
		"empty document id": func(t *testing.T) error {
			cl := mustClient(t)
			_, err := cl.UploadVersion(t.Context(), "", testSignature(t), strings.NewReader("hello"))
			return err
		},
		"unreachable host": func(t *testing.T) error {
			srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
			host := srv.URL
			srv.Close()
			cl, err := client.New("tok", client.WithHost(host))
			if err != nil {
				t.Fatalf("unexpected error building client: %v", err)
			}
			_, err = cl.Upload(t.Context(), testSignature(t), strings.NewReader("hello"))
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			err := produce(t)
			if err == nil {
				t.Fatal("expected a failure, got none")
			}
			matched := 0
			for _, sentinel := range sentinels {
				if errors.Is(err, sentinel) {
					matched++
				}
			}
			if matched != 1 {
				t.Errorf("error %v matches %d sentinels, want exactly 1", err, matched)
			}
			if _, ok := errors.AsType[*client.Error](err); !ok {
				t.Errorf("error %v is not a *client.Error", err)
			}
		})
	}
}

func mustClient(t *testing.T) *client.Client {
	t.Helper()
	srv, _ := refusingServer(t)
	cl, err := client.New("secret-token", client.WithHost(srv.URL))
	if err != nil {
		t.Fatalf("unexpected error building client: %v", err)
	}
	return cl
}
