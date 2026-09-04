package bonum

import (
	"net/http"
	"strconv"

	"github.com/techpartners-asia/bonum-go/types"
)

// ListPaymentPlans returns the merchant's subscription plans (managed on the merchant portal).
func (c *Client) ListPaymentPlans() (*types.ListPaymentPlansResponse, error) {
	req, err := c.authedRequest()
	if err != nil {
		return nil, err
	}
	var out types.ListPaymentPlansResponse
	if err := c.do(req, http.MethodGet, mpayPath+"/values/payment-plans", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Subscribe enrols a card token in a payment plan. If today matches the plan's cycleValue
// (or PayNow is set) the first charge happens immediately.
func (c *Client) Subscribe(cardToken string, input types.SubscribeInput) (*types.SubscribeResponse, error) {
	req, err := c.authedRequest()
	if err != nil {
		return nil, err
	}
	var out types.SubscribeResponse
	if err := c.do(req.SetHeader(cardTokenHeader, cardToken).SetBody(input), http.MethodPost, mpayPath+"/subscriptions/subscribe", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetSubscriptions lists the subscriptions attached to a card token.
func (c *Client) GetSubscriptions(cardToken string) (*types.GetSubscriptionsResponse, error) {
	req, err := c.authedRequest()
	if err != nil {
		return nil, err
	}
	var out types.GetSubscriptionsResponse
	if err := c.do(req.SetHeader(cardTokenHeader, cardToken), http.MethodGet, mpayPath+"/subscriptions", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ChangeSubscriptionTokenNew moves a subscription onto a brand-new card by starting a
// tokenization flow. Redirect the customer to FollowUpLink.
func (c *Client) ChangeSubscriptionTokenNew(subscriptionID int64, input types.ChangeSubscriptionTokenInput) (*types.CreateCardTokenResponse, error) {
	req, err := c.authedRequest()
	if err != nil {
		return nil, err
	}
	var out types.CreateCardTokenResponse
	if err := c.do(req.SetPathParam("id", strconv.FormatInt(subscriptionID, 10)).SetBody(input),
		http.MethodPut, mpayPath+"/subscriptions/{id}/change/create-new-token", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ChangeSubscriptionTokenExisting moves a subscription onto an already stored card token.
func (c *Client) ChangeSubscriptionTokenExisting(subscriptionID int64, cardToken string) (*types.SubscribeResponse, error) {
	req, err := c.authedRequest()
	if err != nil {
		return nil, err
	}
	var out types.SubscribeResponse
	if err := c.do(req.SetHeader(cardTokenHeader, cardToken).SetPathParam("id", strconv.FormatInt(subscriptionID, 10)),
		http.MethodPut, mpayPath+"/subscriptions/{id}/change", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Unsubscribe cancels a subscription; the already scheduled next billing still runs.
func (c *Client) Unsubscribe(subscriptionID, planID int64) (*types.SubscriptionActionResponse, error) {
	return c.subscriptionDelete(subscriptionID, planID, mpayPath+"/subscriptions/{id}")
}

// DeleteSubscription cancels a subscription immediately; no further billing is created.
func (c *Client) DeleteSubscription(subscriptionID, planID int64) (*types.SubscriptionActionResponse, error) {
	return c.subscriptionDelete(subscriptionID, planID, mpayPath+"/subscriptions/{id}/delete")
}

func (c *Client) subscriptionDelete(subscriptionID, planID int64, path string) (*types.SubscriptionActionResponse, error) {
	req, err := c.authedRequest()
	if err != nil {
		return nil, err
	}
	var out types.SubscriptionActionResponse
	if err := c.do(req.SetPathParam("id", strconv.FormatInt(subscriptionID, 10)).SetBody(map[string]int64{"planId": planID}),
		http.MethodDelete, path, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ExecuteSubscriptionPaymentSandbox triggers a billing run on demand. Sandbox only.
func (c *Client) ExecuteSubscriptionPaymentSandbox(subscriptionID int64) (*types.SubscriptionActionResponse, error) {
	req, err := c.authedRequest()
	if err != nil {
		return nil, err
	}
	var out types.SubscriptionActionResponse
	if err := c.do(req.SetPathParam("id", strconv.FormatInt(subscriptionID, 10)),
		http.MethodPut, mpayPath+"/subscriptions/{id}/execute", &out); err != nil {
		return nil, err
	}
	return &out, nil
}
