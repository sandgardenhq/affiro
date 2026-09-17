package client_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sandgardenhq/affiro/client"
)

func answering(t *testing.T, body string) *client.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// A large body leaves this write short: the client stops reading at its own cap.
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	cl, err := client.New("secret-token", client.WithHost(srv.URL))
	if err != nil {
		t.Fatalf("unexpected error building client: %v", err)
	}
	return cl
}

func TestUnrecognisedVerdictIsPassedThrough(t *testing.T) {
	t.Parallel()
	cl := answering(t, `{"id":"sdc-1","humanWritten":"almost certainly"}`)

	analysis, err := cl.Upload(t.Context(), testSignature(t), strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if analysis.HumanWritten != "almost certainly" {
		t.Errorf("got verdict %q, want it passed through unchanged", analysis.HumanWritten)
	}
	if analysis.HumanWritten == client.HumanWrittenUnknown {
		t.Error("an unrecognised verdict collapsed to unknown")
	}
}

func TestEveryDocumentedVerdictDecodes(t *testing.T) {
	t.Parallel()
	for _, verdict := range []client.HumanWritten{
		client.HumanWrittenLikely,
		client.HumanWrittenPossible,
		client.HumanWrittenUnlikely,
		client.HumanWrittenVeryUnlikely,
		client.HumanWrittenUnknown,
	} {
		t.Run(string(verdict), func(t *testing.T) {
			t.Parallel()
			cl := answering(t, `{"id":"sdc-1","humanWritten":"`+string(verdict)+`"}`)
			analysis, err := cl.Upload(t.Context(), testSignature(t), strings.NewReader("hello"))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if analysis.HumanWritten != verdict {
				t.Errorf("got verdict %q, want %q", analysis.HumanWritten, verdict)
			}
		})
	}
}

func TestSuccessWithAnUnreadableBodyIsUnavailable(t *testing.T) {
	t.Parallel()
	cl := answering(t, "this is not json")

	_, err := cl.Upload(t.Context(), testSignature(t), strings.NewReader("hello"))
	if !errors.Is(err, client.ErrUnavailable) {
		t.Fatalf("got error %v, want one matching %v", err, client.ErrUnavailable)
	}
	if errors.Is(err, client.ErrUnauthorized) || errors.Is(err, client.ErrInvalidDocument) {
		t.Error("an unreadable reply reads as an API key or document problem")
	}
}

func TestOversizedReplyIsRefusedRatherThanTruncated(t *testing.T) {
	t.Parallel()
	// Valid JSON then padding: truncating at the cap would still decode, so only refusing an
	// over-cap reply outright fails this.
	padding := strings.Repeat(" ", 2<<20)
	cl := answering(t, `{"id":"sdc-1","humanWritten":"likely"}`+padding)

	analysis, err := cl.Upload(t.Context(), testSignature(t), strings.NewReader("hello"))
	if err == nil {
		t.Fatalf("an over-cap reply decoded as %+v instead of failing", analysis)
	}
	if !errors.Is(err, client.ErrUnavailable) {
		t.Errorf("got error %v, want one matching %v", err, client.ErrUnavailable)
	}
}
