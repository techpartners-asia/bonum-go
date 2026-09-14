package bonum

import "github.com/techpartners-asia/bonum-go/internal/gateway/domain"

// Sentinel errors shared by every aggregate. Match them with errors.Is; the concrete
// *APIError / *ValidationError is still available through errors.As. Aggregate-specific
// errors (ErrDeclined in card.go, ErrBadChecksum / ErrUnknownEvent in webhook.go) live
// alongside that aggregate's other types.
var (
	ErrInvalidInput = domain.ErrInvalidInput // local validation failed, or Bonum answered 400
	ErrUnauthorized = domain.ErrUnauthorized // 401 / 403: bad AppSecret, terminal or token
	ErrNotFound     = domain.ErrNotFound     // 404
	ErrRateLimited  = domain.ErrRateLimited  // 429
)

type (
	// APIError is returned when Bonum answers with a non-2xx status.
	APIError = domain.APIError
	// ValidationError is returned before any network call when an input violates an invariant.
	ValidationError = domain.ValidationError
)
