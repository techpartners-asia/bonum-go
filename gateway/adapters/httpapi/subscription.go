package httpapi

import (
	"context"
	"net/http"
	"strconv"

	"github.com/techpartners-asia/bonum-go/gateway/domain/card"
	"github.com/techpartners-asia/bonum-go/gateway/domain/subscription"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

var _ ports.SubscriptionAPI = (*Client)(nil)

func subscriptionID(id int64) reqOpt { return pathParam("id", strconv.FormatInt(id, 10)) }

func (c *Client) Plans(ctx context.Context) ([]subscription.PaymentPlan, error) {
	out, err := callEnveloped[[]subscription.PaymentPlan](ctx, c, http.MethodGet, mpayPath+"/values/payment-plans")
	if err != nil {
		return nil, err
	}
	return *out, nil
}

func (c *Client) Subscribe(ctx context.Context, cardTok string, in subscription.SubscribeInput) (*subscription.Subscription, error) {
	return callEnveloped[subscription.Subscription](ctx, c, http.MethodPost, mpayPath+"/subscriptions/subscribe", cardToken(cardTok), body(in))
}

func (c *Client) ListSubscriptions(ctx context.Context, cardTok string) ([]subscription.Subscription, error) {
	out, err := callEnveloped[[]subscription.Subscription](ctx, c, http.MethodGet, mpayPath+"/subscriptions", cardToken(cardTok))
	if err != nil {
		return nil, err
	}
	return *out, nil
}

func (c *Client) ChangeCardByTokenizing(ctx context.Context, id int64, in subscription.ChangeCardInput) (*card.Tokenization, error) {
	return call[card.Tokenization](ctx, c, http.MethodPut, mpayPath+"/subscriptions/{id}/change/create-new-token", subscriptionID(id), body(in))
}

func (c *Client) ChangeCard(ctx context.Context, id int64, cardTok string) (*subscription.Subscription, error) {
	return callEnveloped[subscription.Subscription](ctx, c, http.MethodPut, mpayPath+"/subscriptions/{id}/change", subscriptionID(id), cardToken(cardTok))
}

func (c *Client) Unsubscribe(ctx context.Context, id, planID int64) error {
	return callAction(ctx, c, http.MethodDelete, mpayPath+"/subscriptions/{id}", subscriptionID(id), body(map[string]int64{"planId": planID}))
}

func (c *Client) DeleteSubscription(ctx context.Context, id, planID int64) error {
	return callAction(ctx, c, http.MethodDelete, mpayPath+"/subscriptions/{id}/delete", subscriptionID(id), body(map[string]int64{"planId": planID}))
}
