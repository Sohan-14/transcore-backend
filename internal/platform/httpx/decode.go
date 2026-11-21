package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"

	"github.com/sohan-14/transcore-backend/internal/apperr"
)

// DecodeJSON decodes exactly one JSON value from the body into dst.
// Unknown fields are ignored (forward-compatible clients).
// Writes nothing; on failure returns an *apperr.Error for WriteError.
// Only call it on requests that carry a body (POST/PUT/PATCH).
// ===============================================================
func DecodeJSON(r *http.Request, dst any) error {
	mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mt != "application/json" {
		return apperr.New(apperr.KindUnsupportedMedia, "unsupported_media_type",
			"Content-Type must be application/json")
	}

	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		return decodeError(err)
	}

	// Reject anything after the first value.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if _, isMax := errors.AsType[*http.MaxBytesError](err); isMax {
			return apperr.New(apperr.KindTooLarge, "request_too_large", "request body too large")
		}
		return apperr.New(apperr.KindInvalid, "invalid_json",
			"request body must contain a single JSON value")
	}
	return nil
}

func decodeError(err error) error {
	if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
		return apperr.New(apperr.KindTooLarge, "request_too_large", "request body too large")
	}
	switch {
	case errors.Is(err, io.EOF):
		return apperr.New(apperr.KindInvalid, "invalid_json", "request body is required")
	case errors.Is(err, io.ErrUnexpectedEOF):
		return apperr.New(apperr.KindInvalid, "invalid_json", "request body is not valid JSON")
	}
	if e, ok := errors.AsType[*json.SyntaxError](err); ok {
		return apperr.New(apperr.KindInvalid, "invalid_json",
			fmt.Sprintf("malformed JSON at position %d", e.Offset))
	}
	if e, ok := errors.AsType[*json.UnmarshalTypeError](err); ok {
		if e.Field != "" {
			return apperr.New(apperr.KindInvalid, "invalid_json",
				fmt.Sprintf("invalid value for field %q", e.Field))
		}
		return apperr.New(apperr.KindInvalid, "invalid_json", "invalid JSON value")
	}
	return apperr.New(apperr.KindInvalid, "invalid_json", "request body is not valid JSON")
}
