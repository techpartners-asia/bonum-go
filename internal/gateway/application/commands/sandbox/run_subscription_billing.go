package sandbox

import (
	"context"

	"github.com/techpartners-asia/bonum-go/internal/gateway/domain"
	"github.com/techpartners-asia/bonum-go/internal/gateway/ports"
)

// RunSubscriptionBillingCommand triggers a billing run for Subscription ID on demand.
type RunSubscriptionBillingCommand struct{ ID int64 }

type RunSubscriptionBillingHandler struct{ api ports.SandboxAPI }

func NewRunSubscriptionBillingHandler(api ports.SandboxAPI) *RunSubscriptionBillingHandler {
	return &RunSubscriptionBillingHandler{api: api}
}

func (h *RunSubscriptionBillingHandler) Handle(ctx context.Context, cmd RunSubscriptionBillingCommand) error {
	if cmd.ID <= 0 {
		return domain.Invalid("id", "required")
	}
	return h.api.RunSubscriptionBilling(ctx, cmd.ID)
}
