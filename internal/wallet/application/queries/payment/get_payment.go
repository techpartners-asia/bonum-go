// Package payment holds the Wallet Payment aggregate's read use cases.
package payment

import (
	"context"

	"github.com/techpartners-asia/bonum-go/internal/wallet/domain"
	"github.com/techpartners-asia/bonum-go/internal/wallet/domain/payment"
	"github.com/techpartners-asia/bonum-go/internal/wallet/ports"
)

// GetPaymentQuery returns the current state of a Wallet Payment by Bonum's paymentId.
type GetPaymentQuery struct{ PaymentID string }

type GetPaymentHandler struct{ api ports.PaymentAPI }

func NewGetPaymentHandler(api ports.PaymentAPI) *GetPaymentHandler {
	return &GetPaymentHandler{api: api}
}

func (h *GetPaymentHandler) Handle(ctx context.Context, q GetPaymentQuery) (*payment.Payment, error) {
	if q.PaymentID == "" {
		return nil, domain.Invalid("paymentID", "required")
	}
	return h.api.GetPayment(ctx, q.PaymentID)
}
