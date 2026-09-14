package subscription

import (
	"context"

	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// DeleteCommand cancels Subscription ID immediately; no further billing is created.
type DeleteCommand struct {
	ID     int64
	PlanID int64
}

type DeleteHandler struct{ api ports.SubscriptionAPI }

func NewDeleteHandler(api ports.SubscriptionAPI) *DeleteHandler { return &DeleteHandler{api: api} }

func (h *DeleteHandler) Handle(ctx context.Context, cmd DeleteCommand) error {
	return h.api.DeleteSubscription(ctx, cmd.ID, cmd.PlanID)
}
