package payment

import (
	"context"

	"github.com/techpartners-asia/bonum-go/internal/wallet/domain"
	"github.com/techpartners-asia/bonum-go/internal/wallet/domain/payment"
	"github.com/techpartners-asia/bonum-go/internal/wallet/ports"
)

// LookupByOrderIDQuery returns the Wallet Payment submitted with your Order ID.
type LookupByOrderIDQuery struct{ OrderID string }

type LookupByOrderIDHandler struct{ api ports.PaymentAPI }

func NewLookupByOrderIDHandler(api ports.PaymentAPI) *LookupByOrderIDHandler {
	return &LookupByOrderIDHandler{api: api}
}

func (h *LookupByOrderIDHandler) Handle(ctx context.Context, q LookupByOrderIDQuery) (*payment.Payment, error) {
	if q.OrderID == "" {
		return nil, domain.Invalid("orderID", "required")
	}
	return h.api.LookupByOrderID(ctx, q.OrderID)
}
