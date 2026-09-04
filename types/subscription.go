package types

import "encoding/json"

type RecurringType string

const (
	RecurringWeekly  RecurringType = "WEEKLY"
	RecurringMonthly RecurringType = "MONTHLY"
	RecurringYearly  RecurringType = "YEARLY"
)

type (
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

	ListPaymentPlansResponse = Envelope[[]PaymentPlan]

	SubscribeInput struct {
		PlanID     int64  `json:"planId"`
		CycleValue int64  `json:"cycleValue"` // 1-7 weekly, 1-31 monthly, 1-366 yearly; ignored when PayNow
		Cycles     *int64 `json:"cycles,omitempty"`
		PayNow     bool   `json:"payNow"`
		CustEmail  string `json:"custEmail,omitempty"`
	}

	Subscription struct {
		SubscriptionID int64       `json:"subscriptionId"`
		SubscribedAt   string      `json:"subscribedAt"`
		CardMask       string      `json:"cardMask"`
		Plan           PaymentPlan `json:"plan"`
		NextBillAt     string      `json:"nextBillAt"`
		LastBilledAt   string      `json:"lastBilledAt"`
		Status         string      `json:"status"`
	}

	SubscribeResponse = Envelope[Subscription]

	// GetSubscriptionsResponse shape is inferred from SubscribeResponse; the collection has no example.
	GetSubscriptionsResponse = Envelope[[]Subscription]

	ChangeSubscriptionTokenInput struct {
		Callback      string `json:"callback"`
		TransactionID string `json:"transactionId"`
		Items         []Item `json:"items,omitempty"`
	}

	// SubscriptionActionResponse covers unsubscribe / delete / execute; the collection has no example body.
	SubscriptionActionResponse = Envelope[json.RawMessage]
)
