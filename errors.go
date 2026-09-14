package bonum

import (
	"github.com/techpartners-asia/bonum-go/gateway/domain"
	"github.com/techpartners-asia/bonum-go/gateway/domain/card"
	"github.com/techpartners-asia/bonum-go/gateway/domain/webhook"
)

// Sentinel errors. Match them with errors.Is; the concrete *APIError / *DeclinedError /
// *ValidationError is still available through errors.As.
var (
	ErrInvalidInput = domain.ErrInvalidInput  // local validation failed, or Bonum answered 400
	ErrUnauthorized = domain.ErrUnauthorized  // 401 / 403: bad AppSecret, terminal or token
	ErrNotFound     = domain.ErrNotFound      // 404
	ErrRateLimited  = domain.ErrRateLimited   // 429
	ErrDeclined     = card.ErrDeclined        // card purchase refused by the bank
	ErrBadChecksum  = webhook.ErrBadChecksum  // webhook checksum mismatch
	ErrUnknownEvent = webhook.ErrUnknownEvent // unknown webhook event type
)

type (
	// APIError is returned when Bonum answers with a non-2xx status.
	APIError = domain.APIError
	// DeclinedError is an APIError whose body carried a FAILED Purchase.
	DeclinedError = card.DeclinedError
	// ValidationError is returned before any network call when an input violates an invariant.
	ValidationError = domain.ValidationError
)
