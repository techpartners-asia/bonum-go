// Package payment holds the Wallet Payment aggregate's write use cases: submitting a wallet
// token.
package payment

import (
	"context"

	"github.com/techpartners-asia/bonum-go/internal/wallet/domain/payment"
	"github.com/techpartners-asia/bonum-go/internal/wallet/ports"
)

// ProcessApplePayCommand submits an Apple Pay token.
type ProcessApplePayCommand = payment.ProcessApplePayInput

// ProcessApplePayHandler submits an Apple Pay token. The response is always PENDING; call
// AwaitPayment (or AwaitURL) to learn the outcome in time to close the wallet sheet.
type ProcessApplePayHandler struct{ api ports.PaymentAPI }

func NewProcessApplePayHandler(api ports.PaymentAPI) *ProcessApplePayHandler {
	return &ProcessApplePayHandler{api: api}
}

func (h *ProcessApplePayHandler) Handle(ctx context.Context, cmd ProcessApplePayCommand) (*payment.ProcessResponse, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}
	return h.api.ProcessApplePay(ctx, cmd)
}
