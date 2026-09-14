package payment

import (
	"context"
	"time"

	"github.com/techpartners-asia/bonum-go/internal/wallet/domain"
	"github.com/techpartners-asia/bonum-go/internal/wallet/domain/payment"
	"github.com/techpartners-asia/bonum-go/internal/wallet/ports"
)

// AwaitURLQuery is AwaitPaymentQuery for the absolute awaitUrl returned by
// ProcessApplePay / ProcessGooglePay.
type AwaitURLQuery struct {
	AwaitURL string
	Timeout  time.Duration
}

type AwaitURLHandler struct{ api ports.PaymentAPI }

func NewAwaitURLHandler(api ports.PaymentAPI) *AwaitURLHandler { return &AwaitURLHandler{api: api} }

func (h *AwaitURLHandler) Handle(ctx context.Context, q AwaitURLQuery) (*payment.AwaitResult, error) {
	if q.AwaitURL == "" {
		return nil, domain.Invalid("awaitURL", "required")
	}
	return h.api.AwaitURL(ctx, q.AwaitURL, clampAwait(q.Timeout))
}
