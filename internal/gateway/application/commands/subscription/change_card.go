package subscription

import (
	"context"

	"github.com/techpartners-asia/bonum-go/internal/gateway/domain"
	"github.com/techpartners-asia/bonum-go/internal/gateway/domain/subscription"
	"github.com/techpartners-asia/bonum-go/internal/gateway/ports"
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
	if cmd.ID <= 0 {
		return nil, domain.Invalid("id", "required")
	}
	if cmd.CardToken == "" {
		return nil, domain.Invalid("cardToken", "required")
	}
	return h.api.ChangeCard(ctx, cmd.ID, cmd.CardToken)
}
