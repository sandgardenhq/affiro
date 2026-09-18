package client_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
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

// receivedRequest is what the test server saw, gathered so a test can compare the whole
// request at once rather than a field at a time.
type receivedRequest struct {
	method     string
	path       string
	authHeader string
	body       string
}

func TestUploadPostsSignatureThenContent(t *testing.T) {
	t.Parallel()
	sig := testSignature(t)

	var got receivedRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.method = r.Method
		got.path = r.URL.Path
		got.authHeader = r.Header.Get("Authorization")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
		}
		got.body = string(body)
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

	want := receivedRequest{
		method:     http.MethodPost,
		path:       "/api/v1/documents",
		authHeader: "Bearer secret-token",
		body:       sig.String() + "\nhello world",
	}
	if got != want {
		t.Errorf("got request %+v, want %+v", got, want)
	}

	wantAnalysis := client.Analysis{
		ID:           "sdc-1",
		DocumentID:   "doc-1",
		HumanWritten: client.HumanWrittenLikely,
		Descriptions: []string{"a"},
	}
	// Analysis carries a slice, so it is not comparable with ==.
	if !reflect.DeepEqual(analysis, wantAnalysis) {
		t.Errorf("got analysis %+v, want %+v", analysis, wantAnalysis)
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
