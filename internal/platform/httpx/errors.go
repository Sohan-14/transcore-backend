package httpx

import (
	"errors"
	"net/http"

	"github.com/sohan-14/transcore-backend/internal/apperr"
	"github.com/sohan-14/transcore-backend/internal/platform/requestid"
)

// WriteError is the only error renderer in the app.
// Unknown errors become a generic 500; causes never reach the client.
// ===============================================================
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	e, ok := apperr.As(err)
	if !ok {
		if _, isMax := errors.AsType[*http.MaxBytesError](err); isMax {
			e = apperr.New(apperr.KindTooLarge, "request_too_large", "request body too large")
		} else {
			e = apperr.Internal(err)
		}
	}

	status := statusFor(e.Kind)
	reqID := requestid.FromContext(r.Context())

	body := errorBody{
		Code:      e.Code,
		Message:   e.Message,
		RequestID: reqID,
	}

	for _, f := range e.Fields {
		body.Fields = append(body.Fields, wireField{Field: f.Name, Issue: f.Issue})
	}
	JSON(w, status, errorEnvelope{Error: body})
}

func statusFor(kind apperr.Kind) int {
	switch kind {
	case apperr.KindInvalid:
		return http.StatusBadRequest
	case apperr.KindUnauthenticated:
		return http.StatusUnauthorized
	case apperr.KindForbidden:
		return http.StatusForbidden
	case apperr.KindNotFound:
		return http.StatusNotFound
	case apperr.KindConflict:
		return http.StatusConflict
	case apperr.KindTooLarge:
		return http.StatusRequestEntityTooLarge
	case apperr.KindUnsupportedMedia:
		return http.StatusUnsupportedMediaType
	case apperr.KindRateLimited:
		return http.StatusTooManyRequests
	case apperr.KindUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	RequestID string      `json:"request_id,omitempty"`
	Fields    []wireField `json:"fields,omitempty"`
}

type wireField struct {
	Field string `json:"field"`
	Issue string `json:"issue"`
}
