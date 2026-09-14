package payment

import (
	"context"
	"time"

	"github.com/techpartners-asia/bonum-go/wallet/domain"
	"github.com/techpartners-asia/bonum-go/wallet/domain/payment"
	"github.com/techpartners-asia/bonum-go/wallet/ports"
)

// AwaitPaymentQuery blocks until the payment reaches AUTHORIZED or FAILED, or Timeout
// elapses. Timeout 0 uses Bonum's default (25s); anything above payment.MaxAwaitTimeout is
// capped at 28s.
type AwaitPaymentQuery struct {
	PaymentID string
	Timeout   time.Duration
}

type AwaitPaymentHandler struct{ api ports.PaymentAPI }

func NewAwaitPaymentHandler(api ports.PaymentAPI) *AwaitPaymentHandler {
	return &AwaitPaymentHandler{api: api}
}

func (h *AwaitPaymentHandler) Handle(ctx context.Context, q AwaitPaymentQuery) (*payment.AwaitResult, error) {
	if q.PaymentID == "" {
		return nil, domain.Invalid("paymentID", "required")
	}
	return h.api.AwaitPayment(ctx, q.PaymentID, clampAwait(q.Timeout))
}

// clampAwait applies the domain rule: non-positive means server default, never above the cap.
// Defined once here; await_url.go in this same package reuses it.
func clampAwait(d time.Duration) time.Duration {
	switch {
	case d <= 0:
		return 0
	case d > payment.MaxAwaitTimeout:
		return payment.MaxAwaitTimeout
	}
	return d
}
