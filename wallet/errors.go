package wallet

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Sentinel errors. Match them with errors.Is; the concrete *APIError / *ValidationError is
// still available through errors.As.
var (
	ErrInvalidInput      = errors.New("bonum wallet: invalid input") // local validation failed, or Bonum answered 400
	ErrUnauthorized      = errors.New("bonum wallet: unauthorized")  // 401: missing or inactive merchant key
	ErrNotFound          = errors.New("bonum wallet: not found")     // 404: unknown payment or order
	ErrRateLimited       = errors.New("bonum wallet: rate limited")  // 429
	ErrMissingSignature  = errors.New("bonum wallet: missing or malformed signature headers")
	ErrTimestampExpired  = errors.New("bonum wallet: webhook timestamp outside replay tolerance")
	ErrSignatureMismatch = errors.New("bonum wallet: webhook signature mismatch")
)

// APIError is returned when Bonum answers with a non-2xx status.
type APIError struct {
	StatusCode int
	Message    string // e.g. "order_id is required"
	ErrorText  string // e.g. "Bad Request"
	Body       string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("bonum wallet: %d %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("bonum wallet: %d %s", e.StatusCode, e.Body)
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
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Error      string `json:"error"`
}

func newAPIError(status int, body string) error {
	e := &APIError{StatusCode: status, Body: body}
	var eb errorBody
	if json.Unmarshal([]byte(body), &eb) == nil {
		e.Message = eb.Message
		e.ErrorText = eb.Error
	}
	return e
}

// ValidationError is returned before any network call when an input violates an invariant.
type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("bonum wallet: invalid %s: %s", e.Field, e.Reason)
}
func (e *ValidationError) Is(target error) bool { return target == ErrInvalidInput }

func invalid(field, reason string) error { return &ValidationError{Field: field, Reason: reason} }
