package payment

import (
	"context"

	"github.com/techpartners-asia/bonum-go/wallet/domain/payment"
	"github.com/techpartners-asia/bonum-go/wallet/ports"
)

// ProcessGooglePayCommand submits a Google Pay token string.
type ProcessGooglePayCommand = payment.ProcessGooglePayInput

// ProcessGooglePayHandler submits a Google Pay token string. See ProcessApplePayHandler for
// the result model.
type ProcessGooglePayHandler struct{ api ports.PaymentAPI }

func NewProcessGooglePayHandler(api ports.PaymentAPI) *ProcessGooglePayHandler {
	return &ProcessGooglePayHandler{api: api}
}

func (h *ProcessGooglePayHandler) Handle(ctx context.Context, cmd ProcessGooglePayCommand) (*payment.ProcessResponse, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}
	return h.api.ProcessGooglePay(ctx, cmd)
}
