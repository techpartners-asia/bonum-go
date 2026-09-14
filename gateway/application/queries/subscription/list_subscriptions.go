package subscription

import (
	"context"

	"github.com/techpartners-asia/bonum-go/gateway/domain/subscription"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// ListSubscriptionsQuery returns the Subscriptions attached to CardToken.
type ListSubscriptionsQuery struct{ CardToken string }

type ListSubscriptionsHandler struct{ api ports.SubscriptionAPI }

func NewListSubscriptionsHandler(api ports.SubscriptionAPI) *ListSubscriptionsHandler {
	return &ListSubscriptionsHandler{api: api}
}

func (h *ListSubscriptionsHandler) Handle(ctx context.Context, q ListSubscriptionsQuery) ([]subscription.Subscription, error) {
	return h.api.ListSubscriptions(ctx, q.CardToken)
}
