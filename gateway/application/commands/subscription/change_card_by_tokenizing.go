package subscription

import (
	"context"

	"github.com/techpartners-asia/bonum-go/gateway/domain/card"
	"github.com/techpartners-asia/bonum-go/gateway/domain/subscription"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// ChangeCardByTokenizingCommand moves Subscription ID onto a brand-new card.
type ChangeCardByTokenizingCommand struct {
	ID int64
	subscription.ChangeCardInput
}

// ChangeCardByTokenizingHandler starts a Tokenization for the Subscription's new card.
// Redirect the customer to FollowUpLink.
type ChangeCardByTokenizingHandler struct{ api ports.SubscriptionAPI }

func NewChangeCardByTokenizingHandler(api ports.SubscriptionAPI) *ChangeCardByTokenizingHandler {
	return &ChangeCardByTokenizingHandler{api: api}
}

func (h *ChangeCardByTokenizingHandler) Handle(ctx context.Context, cmd ChangeCardByTokenizingCommand) (*card.Tokenization, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}
	return h.api.ChangeCardByTokenizing(ctx, cmd.ID, cmd.ChangeCardInput)
}
