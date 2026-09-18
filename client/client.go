// Package client uploads signed documents to the affiro API with an API key.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"

	"github.com/sandgardenhq/affiro/asig"
)

// DefaultHost is the API this client talks to unless [WithHost] says otherwise.
const DefaultHost = "https://app.affiro.com"

// Client uploads documents to an API as the user its API key belongs to.
type Client struct {
	http  *http.Client
	token string
	host  string
}

// Option configures a [Client] at construction.
type Option func(*Client)

// WithHost points the client at an API other than [DefaultHost], such as a local server.
func WithHost(host string) Option {
	return func(c *Client) {
		c.host = host
	}
}

// WithHTTPClient supplies an [http.Client] of the caller's own, for a timeout or a proxy.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) {
		c.http = h
	}
}

// New returns a client that presents the given API key on every call.
func New(token string, opts ...Option) (*Client, error) {
	if token == "" {
		return nil, refused(ErrUnauthorized, "no API key was supplied to client.New")
	}
	c := &Client{
		token: token,
		host:  DefaultHost,
		http:  http.DefaultClient,
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(c)
	}
	// A nil http.Client panics inside Do rather than acting like the default.
	if c.http == nil {
		c.http = http.DefaultClient
	}
	// A trailing slash would double against the leading slash of every path below.
	c.host = strings.TrimRight(strings.TrimSpace(c.host), "/")
	if err := usableHost(c.host); err != nil {
		return nil, err
	}
	return c, nil
}

// usableHost rejects a host that would otherwise fail mid-request, reading as an unreachable
// API rather than as the argument mistake it is.
func usableHost(host string) error {
	if host == "" {
		return refused(ErrUnavailable, "no API host was supplied")
	}
	parsed, err := url.Parse(host)
	if err != nil {
		return refused(ErrUnavailable, "API host is not a URL: "+err.Error())
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return refused(ErrUnavailable, "API host needs a scheme and a host, as in "+DefaultHost)
	}
	return nil
}

// Upload stores content as a new document, signed by sig, and returns the API's analysis.
func (c *Client) Upload(ctx context.Context, sig *asig.Asig, content io.Reader) (Analysis, error) {
	return c.upload(ctx, "/api/v1/documents", sig, content)
}

// UploadVersion stores content as a new version of an existing document. documentID is the
// DocumentID of an earlier [Analysis], or an external identifier the caller gave the document.
func (c *Client) UploadVersion(ctx context.Context, documentID string, sig *asig.Asig, content io.Reader) (Analysis, error) {
	// "." and ".." survive PathEscape intact, walking the request off the documents route.
	switch strings.TrimSpace(documentID) {
	case "":
		return Analysis{}, refused(ErrInvalidDocument, "no document id was supplied")
	case ".", "..":
		return Analysis{}, refused(ErrInvalidDocument, "document id "+documentID+" does not name a document")
	}
	return c.upload(ctx, "/api/v1/documents/"+url.PathEscape(documentID), sig, content)
}

func (c *Client) upload(ctx context.Context, path string, sig *asig.Asig, content io.Reader) (Analysis, error) {
	if sig == nil {
		return Analysis{}, refused(ErrInvalidDocument, "no signature was supplied")
	}
	if readerIsNil(content) {
		return Analysis{}, refused(ErrInvalidDocument, "no document content was supplied")
	}
	prefix := sig.String() + "\n"
	reqBody := io.MultiReader(strings.NewReader(prefix), content)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.host+path, reqBody)
	if err != nil {
		return Analysis{}, refused(ErrUnavailable, "could not build the request: "+err.Error())
	}
	// The API cannot hold a chunked body to its size limit: it truncates the document and
	// analyses what is left. A declared length lets it refuse an oversized one instead.
	if length, ok := contentLength(content); ok {
		req.ContentLength = int64(len(prefix)) + length
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	resp, err := c.http.Do(req)
	if err != nil {
		return Analysis{}, unreachable(err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	// One byte past the cap, so an over-cap reply can be told from one that merely fills it.
	replyBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
	if err != nil {
		return Analysis{}, unreachable(err)
	}
	if len(replyBody) > maxResponseSize {
		return Analysis{}, refused(ErrUnavailable, "the reply was larger than an API's ever is")
	}
	if resp.StatusCode != http.StatusOK {
		return Analysis{}, statusError(resp.StatusCode, c.redact(serverMessage(replyBody)))
	}
	var analysis Analysis
	if err := json.Unmarshal(replyBody, &analysis); err != nil {
		return Analysis{}, unreachable(err)
	}
	return analysis, nil
}

// maxResponseSize caps what a reply can cost in memory. Over-cap replies are refused rather than
// truncated, because a truncated reply can still parse and would then be believed.
const maxResponseSize = 1 << 20

// redact removes the API key from anything the API said back. A server that quotes the
// request it received would otherwise put the key into an error the caller prints or logs. Only
// the key as written can be removed; a re-encoded one would survive.
func (c *Client) redact(message string) string {
	return strings.ReplaceAll(message, c.token, "[redacted]")
}

// serverMessage is the API's explanation from an error reply, or the raw body when the
// reply is not that shape.
func serverMessage(body []byte) string {
	var reply struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &reply); err == nil && reply.Error != "" {
		return reply.Error
	}
	return strings.TrimSpace(string(body))
}

func contentLength(content io.Reader) (int64, bool) {
	switch src := content.(type) {
	// Len is the unread remainder, which is what goes on the wire; Size would be the whole thing.
	case *strings.Reader:
		return int64(src.Len()), true
	case *bytes.Reader:
		return int64(src.Len()), true
	case *bytes.Buffer:
		return int64(src.Len()), true
	case *os.File:
		info, err := src.Stat()
		if err != nil || !info.Mode().IsRegular() {
			return 0, false
		}
		at, err := src.Seek(0, io.SeekCurrent)
		// Seeked past the end, the length would go negative; net/http reads that as a body that
		// never arrives.
		if err != nil || at > info.Size() {
			return 0, false
		}
		return info.Size() - at, true
	default:
		return 0, false
	}
}

// readerIsNil covers both no reader and a nil pointer wearing the interface, which is not
// equal to nil and panics on the first method call.
func readerIsNil(content io.Reader) bool {
	if content == nil {
		return true
	}
	value := reflect.ValueOf(content)
	return value.Kind() == reflect.Pointer && value.IsNil()
}
