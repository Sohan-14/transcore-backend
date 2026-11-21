package apperr

import (
	"errors"
	"fmt"
)

// Kind classifies an error independently of the transport.
type Kind string

// Error kinds classify application errors for consistent mapping across transport layers.
const (
	KindInvalid          Kind = "invalid"
	KindUnauthenticated  Kind = "unauthenticated"
	KindForbidden        Kind = "forbidden"
	KindNotFound         Kind = "not_found"
	KindConflict         Kind = "conflict"
	KindTooLarge         Kind = "too_large"
	KindUnsupportedMedia Kind = "unsupported_media"
	KindRateLimited      Kind = "rate_limited"
	KindUnavailable      Kind = "unavailable"
	KindInternal         Kind = "internal"
)

// Field describes one invalid input field.
type Field struct {
	Name  string
	Issue string
}

// Error is the application error type.
type Error struct {
	Kind    Kind
	Code    string
	Message string
	Fields  []Field

	cause error
}

// Error combine the error into one string
func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s: %s: %v", e.Kind, e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("%s: %s: %s", e.Kind, e.Code, e.Message)
}

// Unwrap exposes the underlying cause to errors.Is / errors.As.
func (e *Error) Unwrap() error {
	return e.cause
}

// WithFields attaches field-level details (validation errors).
func (e *Error) WithFields(fields ...Field) *Error {
	if e == nil {
		return nil
	}
	cp := *e
	cp.Fields = append(append([]Field(nil), e.Fields...), fields...)
	return &cp
}

// New creates an error of the given kind.
func New(kind Kind, code, message string) *Error {
	return &Error{Kind: kind, Code: code, Message: message}
}

// Wrap creates an error of the given kind that keeps cause for logging.
func Wrap(cause error, kind Kind, code, message string) *Error {
	return &Error{Kind: kind, Code: code, Message: message, cause: cause}
}

// Internal wraps an unexpected failure. The client only sees a generic message.
func Internal(cause error) *Error {
	return Wrap(cause, KindInternal, "internal_error", "internal server error")
}

// As returns the *Error in err's chain, if any.
func As(err error) (*Error, bool) {
	if e, ok := errors.AsType[*Error](err); ok {
		return e, true
	}
	return nil, false
}

// KindOf returns the Kind of err, or KindInternal if err is not an *Error.
func KindOf(err error) Kind {
	if e, ok := errors.AsType[*Error](err); ok {
		return e.Kind
	}
	return KindInternal
}
