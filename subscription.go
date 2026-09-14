package bonum

import (
	"context"
	"net/http"
	"strconv"
)

type RecurringType string

const (
	RecurringWeekly  RecurringType = "WEEKLY"
	RecurringMonthly RecurringType = "MONTHLY"
	RecurringYearly  RecurringType = "YEARLY"
)

type (
	// PaymentPlan is a recurring billing template managed on the merchant portal.
	PaymentPlan struct {
		PlanID        int64         `json:"planId"`
		Name          string        `json:"name"`
		Remark        string        `json:"remark"`
		CreatedAt     string        `json:"createdAt"`
		RecurringType RecurringType `json:"recurringType"`
		Amount        float64       `json:"amount"`
		Status        string        `json:"status"`
		CardCount     int64         `json:"cardCount"`
		RetryCount    int64         `json:"retryCount"`
	}

	SubscribeInput struct {
		PlanID     int64  `json:"planId"`
		CycleValue int64  `json:"cycleValue"` // 1-7 weekly, 1-31 monthly, 1-366 yearly; ignored when PayNow
		Cycles     *int64 `json:"cycles,omitempty"`
		PayNow     bool   `json:"payNow"`
		CustEmail  string `json:"custEmail,omitempty"`
	}

	// Subscription is a Card Token enrolled in a Payment Plan.
	Subscription struct {
		SubscriptionID int64       `json:"subscriptionId"`
		SubscribedAt   string      `json:"subscribedAt"`
		CardMask       string      `json:"cardMask"`
		Plan           PaymentPlan `json:"plan"`
		NextBillAt     string      `json:"nextBillAt"`
		LastBilledAt   string      `json:"lastBilledAt"`
		Status         string      `json:"status"`
	}

	ChangeCardInput struct {
		Callback      string `json:"callback"`
		TransactionID string `json:"transactionId"`
		Items         []Item `json:"items,omitempty"`
	}
)

func (in SubscribeInput) validate() error {
	switch {
	case in.PlanID <= 0:
		return invalid("PlanID", "required")
	case !in.PayNow && (in.CycleValue < 1 || in.CycleValue > 366):
		return invalid("CycleValue", "must be 1-7 weekly, 1-31 monthly or 1-366 yearly")
	}
	return nil
}

func (in ChangeCardInput) validate() error {
	switch {
	case in.Callback == "":
		return invalid("Callback", "required")
	case in.TransactionID == "":
		return invalid("TransactionID", "required")
	}
	return nil
}

func subscriptionID(id int64) reqOpt { return pathParam("id", strconv.FormatInt(id, 10)) }

// SubscriptionService is the Subscription aggregate: recurring charges on a Card Token.
type SubscriptionService struct{ c *Client }

// Plans returns the Terminal's Payment Plans.
func (s *SubscriptionService) Plans(ctx context.Context) ([]PaymentPlan, error) {
	out, err := callEnveloped[[]PaymentPlan](ctx, s.c, http.MethodGet, mpayPath+"/values/payment-plans")
	if err != nil {
		return nil, err
	}
	return *out, nil
}

// Subscribe enrols a Card Token in a Payment Plan. If today matches CycleValue (or PayNow
// is set) the first charge happens immediately.
func (s *SubscriptionService) Subscribe(ctx context.Context, cardTok string, in SubscribeInput) (*Subscription, error) {
	if err := in.validate(); err != nil {
		return nil, err
	}
	return callEnveloped[Subscription](ctx, s.c, http.MethodPost, mpayPath+"/subscriptions/subscribe", cardToken(cardTok), body(in))
}

// List returns the Subscriptions attached to a Card Token.
func (s *SubscriptionService) List(ctx context.Context, cardTok string) ([]Subscription, error) {
	out, err := callEnveloped[[]Subscription](ctx, s.c, http.MethodGet, mpayPath+"/subscriptions", cardToken(cardTok))
	if err != nil {
		return nil, err
	}
	return *out, nil
}

// ChangeCardByTokenizing moves a Subscription onto a brand-new card by starting a
// Tokenization. Redirect the customer to FollowUpLink.
func (s *SubscriptionService) ChangeCardByTokenizing(ctx context.Context, id int64, in ChangeCardInput) (*Tokenization, error) {
	if err := in.validate(); err != nil {
		return nil, err
	}
	return call[Tokenization](ctx, s.c, http.MethodPut, mpayPath+"/subscriptions/{id}/change/create-new-token", subscriptionID(id), body(in))
}

// ChangeCard moves a Subscription onto an already stored Card Token.
func (s *SubscriptionService) ChangeCard(ctx context.Context, id int64, cardTok string) (*Subscription, error) {
	return callEnveloped[Subscription](ctx, s.c, http.MethodPut, mpayPath+"/subscriptions/{id}/change", subscriptionID(id), cardToken(cardTok))
}

// Unsubscribe cancels a Subscription; the already scheduled next billing still runs.
func (s *SubscriptionService) Unsubscribe(ctx context.Context, id, planID int64) error {
	return callAction(ctx, s.c, http.MethodDelete, mpayPath+"/subscriptions/{id}", subscriptionID(id), body(map[string]int64{"planId": planID}))
}

// Delete cancels a Subscription immediately; no further billing is created.
func (s *SubscriptionService) Delete(ctx context.Context, id, planID int64) error {
	return callAction(ctx, s.c, http.MethodDelete, mpayPath+"/subscriptions/{id}/delete", subscriptionID(id), body(map[string]int64{"planId": planID}))
}
