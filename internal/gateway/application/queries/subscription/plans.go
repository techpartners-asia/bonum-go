// Package subscription holds the Subscription aggregate's read use cases.
package subscription

import (
	"context"

	"github.com/techpartners-asia/bonum-go/internal/gateway/domain/subscription"
	"github.com/techpartners-asia/bonum-go/internal/gateway/ports"
)

// PlansHandler returns the Terminal's Payment Plans.
type PlansHandler struct{ api ports.SubscriptionAPI }

func NewPlansHandler(api ports.SubscriptionAPI) *PlansHandler { return &PlansHandler{api: api} }

func (h *PlansHandler) Handle(ctx context.Context) ([]subscription.PaymentPlan, error) {
	return h.api.Plans(ctx)
}
