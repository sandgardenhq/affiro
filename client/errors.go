package client

import (
	"errors"
	"net/http"
	"strconv"
)

// The three ways a call can fail. Match one with [errors.Is] rather than reading an error
// message: every error this package returns matches exactly one of them.
var (
	// ErrUnauthorized means the API key was refused: unknown, revoked, or not allowed to write
	// here. It is also what [New] returns when given no key at all.
	ErrUnauthorized = errors.New("affiro: API key rejected")
	// ErrInvalidDocument means the document could not be stored: the API refused to keep
	// it, or this client refused to send it. [Error.StatusCode] is 0 in the second case.
	ErrInvalidDocument = errors.New("affiro: document rejected")
	// ErrUnavailable means the API could not be reached or could not answer. The call is
	// worth retrying; the other two are not.
	ErrUnavailable = errors.New("affiro: API unavailable")
)

// Error is the concrete type behind every failure here. It unwraps to exactly one of the three
// sentinels, and to the underlying transport error when there was one.
type Error struct {
	kind  error
	cause error

	Message    string
	StatusCode int
}

func (e *Error) Error() string {
	if e.StatusCode == 0 {
		return e.kind.Error() + ": " + e.Message
	}
	return e.kind.Error() + " (" + strconv.Itoa(e.StatusCode) + "): " + e.Message
}

// Unwrap reports the kind of failure, and what went wrong on the way when there was a cause.
func (e *Error) Unwrap() []error {
	if e.cause == nil {
		return []error{e.kind}
	}
	return []error{e.kind, e.cause}
}

// refused reports an argument this client would not send; no API saw it, so no status.
func refused(kind error, message string) *Error {
	return &Error{
		Message: message,
		kind:    kind,
	}
}

// unreachable reports a failure to get an answer at all.
func unreachable(cause error) *Error {
	return &Error{
		Message: cause.Error(),
		kind:    ErrUnavailable,
		cause:   cause,
	}
}

// statusError classifies an answer the API did give. A 4xx that is not about the API key
// is about the document, since these routes take nothing else from the caller.
func statusError(status int, message string) *Error {
	kind := ErrUnavailable
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		kind = ErrUnauthorized
	case status >= http.StatusBadRequest && status < http.StatusInternalServerError:
		kind = ErrInvalidDocument
	}
	if message == "" {
		message = http.StatusText(status)
	}
	return &Error{
		StatusCode: status,
		Message:    message,
		kind:       kind,
	}
}
