package wallet

import "github.com/techpartners-asia/bonum-go/wallet/domain"

// Sentinel errors shared by every aggregate. Match them with errors.Is; the concrete
// *APIError / *ValidationError is still available through errors.As. Webhook-specific
// errors (ErrMissingSignature, ErrTimestampExpired, ErrSignatureMismatch) live in webhook.go.
var (
	ErrInvalidInput = domain.ErrInvalidInput // local validation failed, or Bonum answered 400
	ErrUnauthorized = domain.ErrUnauthorized // 401: missing or inactive merchant key
	ErrNotFound     = domain.ErrNotFound     // 404: unknown payment or order
	ErrRateLimited  = domain.ErrRateLimited  // 429
)

type (
	// APIError is returned when Bonum answers with a non-2xx status.
	APIError = domain.APIError
	// ValidationError is returned before any network call when an input violates an invariant.
	ValidationError = domain.ValidationError
)
