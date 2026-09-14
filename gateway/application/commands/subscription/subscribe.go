// Package subscription holds the Subscription aggregate's write use cases.
package subscription

import (
	"context"

	"github.com/techpartners-asia/bonum-go/gateway/domain/subscription"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// SubscribeCommand enrols CardToken in a Payment Plan.
type SubscribeCommand struct {
	CardToken string
	subscription.SubscribeInput
}

// SubscribeHandler enrols a Card Token in a Payment Plan. If today matches CycleValue (or
// PayNow is set) the first charge happens immediately.
type SubscribeHandler struct{ api ports.SubscriptionAPI }

func NewSubscribeHandler(api ports.SubscriptionAPI) *SubscribeHandler {
	return &SubscribeHandler{api: api}
}

func (h *SubscribeHandler) Handle(ctx context.Context, cmd SubscribeCommand) (*subscription.Subscription, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}
	return h.api.Subscribe(ctx, cmd.CardToken, cmd.SubscribeInput)
}
