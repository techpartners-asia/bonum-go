package card

import (
	"context"

	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// ReverseCommand rolls back a Purchase identified by the merchant TransactionID.
type ReverseCommand struct {
	CardToken     string
	TransactionID string
}

type ReverseHandler struct{ api ports.CardAPI }

func NewReverseHandler(api ports.CardAPI) *ReverseHandler { return &ReverseHandler{api: api} }

func (h *ReverseHandler) Handle(ctx context.Context, cmd ReverseCommand) error {
	return h.api.Reverse(ctx, cmd.CardToken, cmd.TransactionID)
}
