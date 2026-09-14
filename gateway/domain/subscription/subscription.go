// Package subscription is the Subscription aggregate: a Card Token enrolled in a Payment Plan.
package subscription

import (
	"github.com/techpartners-asia/bonum-go/gateway/domain"
	"github.com/techpartners-asia/bonum-go/gateway/domain/checkout"
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
		Callback      string          `json:"callback"`
		TransactionID string          `json:"transactionId"`
		Items         []checkout.Item `json:"items,omitempty"`
	}
)

// Validate enforces the Subscription invariants before any network call.
func (in SubscribeInput) Validate() error {
	switch {
	case in.PlanID <= 0:
		return domain.Invalid("PlanID", "required")
	case !in.PayNow && (in.CycleValue < 1 || in.CycleValue > 366):
		return domain.Invalid("CycleValue", "must be 1-7 weekly, 1-31 monthly or 1-366 yearly")
	}
	return nil
}

// Validate enforces the card-change invariants before any network call.
func (in ChangeCardInput) Validate() error {
	switch {
	case in.Callback == "":
		return domain.Invalid("Callback", "required")
	case in.TransactionID == "":
		return domain.Invalid("TransactionID", "required")
	}
	return nil
}
