package subscription

import (
	"context"

	"github.com/techpartners-asia/bonum-go/gateway/domain/subscription"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// ChangeCardCommand moves Subscription ID onto an already stored Card Token.
type ChangeCardCommand struct {
	ID        int64
	CardToken string
}

type ChangeCardHandler struct{ api ports.SubscriptionAPI }

func NewChangeCardHandler(api ports.SubscriptionAPI) *ChangeCardHandler {
	return &ChangeCardHandler{api: api}
}

func (h *ChangeCardHandler) Handle(ctx context.Context, cmd ChangeCardCommand) (*subscription.Subscription, error) {
	return h.api.ChangeCard(ctx, cmd.ID, cmd.CardToken)
}
