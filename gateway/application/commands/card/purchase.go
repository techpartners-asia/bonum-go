package card

import (
	"context"

	"github.com/techpartners-asia/bonum-go/gateway/domain/card"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// PurchaseCommand charges CardToken for the amount and details in the embedded PurchaseInput.
type PurchaseCommand struct {
	CardToken string
	card.PurchaseInput
}

// PurchaseHandler charges a Card Token. Under load Bonum may answer with Status QUEUED; the
// final result then arrives as a TokenPaymentEvent. A bank refusal is returned as
// *card.DeclinedError.
type PurchaseHandler struct{ api ports.CardAPI }

func NewPurchaseHandler(api ports.CardAPI) *PurchaseHandler { return &PurchaseHandler{api: api} }

func (h *PurchaseHandler) Handle(ctx context.Context, cmd PurchaseCommand) (*card.Purchase, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}
	return h.api.Purchase(ctx, cmd.CardToken, cmd.PurchaseInput)
}
