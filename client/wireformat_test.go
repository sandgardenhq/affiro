package client_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sandgardenhq/affiro/asig"
	"github.com/sandgardenhq/affiro/client"
)

func TestRequestBodyParsesTheWayTheAPIParsesIt(t *testing.T) {
	t.Parallel()
	sig := testSignature(t)
	const content = "the text that was typed\nover more than one line\n"

	var received []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		received, err = io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
		}
		_, _ = io.WriteString(w, `{"id":"sdc-1"}`)
	}))
	defer srv.Close()

	cl, err := client.New("secret-token", client.WithHost(srv.URL))
	if err != nil {
		t.Fatalf("unexpected error building client: %v", err)
	}
	if _, err := cl.Upload(t.Context(), sig, strings.NewReader(content)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// asig.Read is what the API itself parses an upload with.
	parsed, remaining, err := asig.Read(strings.NewReader(string(received)))
	if err != nil {
		t.Fatalf("the API could not read the signature off the body: %v", err)
	}
	if parsed.String() != sig.String() {
		t.Errorf("signature survived as %q, want %q", parsed.String(), sig.String())
	}
	rest, err := io.ReadAll(remaining)
	if err != nil {
		t.Fatalf("failed to read the content after the signature: %v", err)
	}
	if string(rest) != content {
		t.Errorf("content survived as %q, want %q", rest, content)
	}
}

func TestRequestDeclaresItsLengthWhenTheContentKnowsIt(t *testing.T) {
	t.Parallel()
	sig := testSignature(t)
	const content = "the text that was typed"

	for name, body := range map[string]io.Reader{
		"a string reader": strings.NewReader(content),
		"a bytes reader":  bytes.NewReader([]byte(content)),
		"a bytes buffer":  bytes.NewBufferString(content),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var gotLength int64
			var gotBodyLen int
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotLength = r.ContentLength
				read, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("failed to read request body: %v", err)
				}
				gotBodyLen = len(read)
				_, _ = io.WriteString(w, `{"id":"sdc-1"}`)
			}))
			defer srv.Close()

			cl, err := client.New("secret-token", client.WithHost(srv.URL))
			if err != nil {
				t.Fatalf("unexpected error building client: %v", err)
			}
			if _, err := cl.Upload(t.Context(), sig, body); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotLength != int64(gotBodyLen) {
				t.Errorf("declared length %d, sent %d bytes", gotLength, gotBodyLen)
			}
		})
	}
}

func TestRequestStaysChunkedWhenTheContentCannotReportItsLength(t *testing.T) {
	t.Parallel()
	var gotLength int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotLength = r.ContentLength
		_, _ = io.Copy(io.Discard, r.Body)
		_, _ = io.WriteString(w, `{"id":"sdc-1"}`)
	}))
	defer srv.Close()

	cl, err := client.New("secret-token", client.WithHost(srv.URL))
	if err != nil {
		t.Fatalf("unexpected error building client: %v", err)
	}
	// A pipe has no length to report, and a caller streaming from one must still work.
	unmeasurable := io.MultiReader(strings.NewReader("streamed"))
	if _, err := cl.Upload(t.Context(), testSignature(t), unmeasurable); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotLength != -1 {
		t.Errorf("got declared length %d, want -1 (chunked)", gotLength)
	}
}
