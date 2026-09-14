package bonum

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Sentinel errors. Match them with errors.Is; the concrete *APIError / *DeclinedError /
// *ValidationError is still available through errors.As.
var (
	ErrInvalidInput = errors.New("bonum: invalid input")     // local validation failed, or Bonum answered 400
	ErrUnauthorized = errors.New("bonum: unauthorized")      // 401 / 403: bad AppSecret, terminal or token
	ErrNotFound     = errors.New("bonum: not found")         // 404
	ErrRateLimited  = errors.New("bonum: rate limited")      // 429
	ErrDeclined     = errors.New("bonum: purchase declined") // card purchase refused by the bank
	ErrBadChecksum  = errors.New("bonum: webhook checksum mismatch")
	ErrUnknownEvent = errors.New("bonum: unknown webhook event type")
)

// APIError is returned when Bonum answers with a non-2xx status.
type APIError struct {
	StatusCode int
	TraceID    string // Quote it to Bonum support
	Message    string
	Body       string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("bonum: %d %s (trace %s)", e.StatusCode, e.Message, e.TraceID)
	}
	return fmt.Sprintf("bonum: %d %s", e.StatusCode, e.Body)
}

// Is maps HTTP status classes onto the sentinel errors.
func (e *APIError) Is(target error) bool {
	switch target {
	case ErrInvalidInput:
		return e.StatusCode == 400
	case ErrUnauthorized:
		return e.StatusCode == 401 || e.StatusCode == 403
	case ErrNotFound:
		return e.StatusCode == 404
	case ErrRateLimited:
		return e.StatusCode == 429
	}
	return false
}

type errorBody struct {
	TraceID string `json:"traceId"`
	Message string `json:"message"`
}

func newAPIError(status int, body string) error {
	e := &APIError{StatusCode: status, Body: body}
	var eb errorBody
	if json.Unmarshal([]byte(body), &eb) == nil {
		e.TraceID = eb.TraceID
		e.Message = eb.Message
	}
	return e
}

// DeclinedError is an APIError whose body carried a FAILED Purchase: the bank refused the
// card. Purchase holds Bonum's record of the attempt (id, respCode, cardStatus).
type DeclinedError struct {
	*APIError
	Purchase Purchase
}

func (e *DeclinedError) Is(target error) bool { return target == ErrDeclined || e.APIError.Is(target) }
func (e *DeclinedError) Unwrap() error        { return e.APIError }

// ValidationError is returned before any network call when an input violates an invariant.
type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("bonum: invalid %s: %s", e.Field, e.Reason)
}
func (e *ValidationError) Is(target error) bool { return target == ErrInvalidInput }

func invalid(field, reason string) error { return &ValidationError{Field: field, Reason: reason} }
