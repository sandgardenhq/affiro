package client_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sandgardenhq/affiro/asig"
	"github.com/sandgardenhq/affiro/client"
)

func testSignature(t *testing.T) *asig.Asig {
	t.Helper()
	sig, err := asig.ParseString("1.AAAAAAAAAAA..AQID")
	if err != nil {
		t.Fatalf("failed to build test signature: %v", err)
	}
	return sig
}

func TestUploadPostsSignatureThenContent(t *testing.T) {
	t.Parallel()
	sig := testSignature(t)

	var gotPath, gotMethod, gotAuth, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotAuth = r.Header.Get("Authorization")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
		}
		gotBody = string(body)
		_, _ = io.WriteString(w, `{"id":"sdc-1","documentId":"doc-1","humanWritten":"likely","descriptions":["a"]}`)
	}))
	defer srv.Close()

	cl, err := client.New("secret-token", client.WithHost(srv.URL))
	if err != nil {
		t.Fatalf("unexpected error building client: %v", err)
	}
	analysis, err := cl.Upload(t.Context(), sig, strings.NewReader("hello world"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for what, pair := range map[string][2]string{
		"method":        {gotMethod, http.MethodPost},
		"path":          {gotPath, "/api/v1/documents"},
		"authorization": {gotAuth, "Bearer secret-token"},
		"body":          {gotBody, sig.String() + "\nhello world"},
		"id":            {analysis.ID, "sdc-1"},
		"document id":   {analysis.DocumentID, "doc-1"},
		"human written": {string(analysis.HumanWritten), string(client.HumanWrittenLikely)},
	} {
		if pair[0] != pair[1] {
			t.Errorf("got %s %q, want %q", what, pair[0], pair[1])
		}
	}
	if len(analysis.Descriptions) != 1 || analysis.Descriptions[0] != "a" {
		t.Errorf("got descriptions %v, want [a]", analysis.Descriptions)
	}
}

func TestUploadVersionPostsToTheDocumentID(t *testing.T) {
	t.Parallel()
	sig := testSignature(t)

	var gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
		}
		gotBody = string(body)
		_, _ = io.WriteString(w, `{"id":"sdc-2","documentId":"doc-1"}`)
	}))
	defer srv.Close()

	cl, err := client.New("secret-token", client.WithHost(srv.URL))
	if err != nil {
		t.Fatalf("unexpected error building client: %v", err)
	}
	analysis, err := cl.UploadVersion(t.Context(), "doc-1", sig, strings.NewReader("a revision"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotPath != "/api/v1/documents/doc-1" {
		t.Errorf("got path %q, want /api/v1/documents/doc-1", gotPath)
	}
	if want := sig.String() + "\na revision"; gotBody != want {
		t.Errorf("got body %q, want %q", gotBody, want)
	}
	if analysis.ID != "sdc-2" {
		t.Errorf("got id %q, want sdc-2", analysis.ID)
	}
}
