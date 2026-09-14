package subscription

import (
	"context"

	"github.com/techpartners-asia/bonum-go/internal/gateway/domain"
	"github.com/techpartners-asia/bonum-go/internal/gateway/ports"
)

// UnsubscribeCommand cancels Subscription ID; the already scheduled next billing still runs.
type UnsubscribeCommand struct {
	ID     int64
	PlanID int64
}

type UnsubscribeHandler struct{ api ports.SubscriptionAPI }

func NewUnsubscribeHandler(api ports.SubscriptionAPI) *UnsubscribeHandler {
	return &UnsubscribeHandler{api: api}
}

func (h *UnsubscribeHandler) Handle(ctx context.Context, cmd UnsubscribeCommand) error {
	if cmd.ID <= 0 {
		return domain.Invalid("id", "required")
	}
	if cmd.PlanID <= 0 {
		return domain.Invalid("planID", "required")
	}
	return h.api.Unsubscribe(ctx, cmd.ID, cmd.PlanID)
}
