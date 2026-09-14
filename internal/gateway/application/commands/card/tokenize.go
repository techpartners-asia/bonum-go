// Package card holds the Card Token aggregate's write use cases: tokenization, purchases
// and reversals.
package card

import (
	"context"

	"github.com/techpartners-asia/bonum-go/internal/gateway/domain/card"
	"github.com/techpartners-asia/bonum-go/internal/gateway/ports"
)

// TokenizeCommand starts a card tokenization flow.
type TokenizeCommand = card.TokenizeInput

// TokenizeHandler starts a card tokenization flow. Redirect the customer to FollowUpLink.
type TokenizeHandler struct{ api ports.CardAPI }

func NewTokenizeHandler(api ports.CardAPI) *TokenizeHandler { return &TokenizeHandler{api: api} }

func (h *TokenizeHandler) Handle(ctx context.Context, cmd TokenizeCommand) (*card.Tokenization, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}
	return h.api.Tokenize(ctx, cmd)
}
