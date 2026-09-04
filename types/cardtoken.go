package types

import "encoding/json"

// TransactionStatus is the outcome of a token purchase.
type TransactionStatus string

const (
	TransactionSuccess TransactionStatus = "SUCCESS"
	TransactionFailed  TransactionStatus = "FAILED"
	TransactionQueued  TransactionStatus = "QUEUED" // Result will arrive later via TOKEN-PAYMENT webhook
)

type (
	// CardTokenPayment charges the card when it is tokenized. Defaults to 0.01 MNT if omitted.
	CardTokenPayment struct {
		Amount float64 `json:"amount"`
	}

	// CardTokenSubscription subscribes the new card token to a payment plan in the same flow.
	CardTokenSubscription struct {
		PlanID     int64  `json:"planId"`
		CycleValue string `json:"cycleValue"` // 1-7 weekly, 1-31 monthly, 1-366 yearly
		Cycles     *int64 `json:"cycles,omitempty"`
		PayNow     bool   `json:"payNow"`
		CustEmail  string `json:"custEmail,omitempty"`
	}

	CreateCardTokenInput struct {
		Callback      string                 `json:"callback"`
		TransactionID string                 `json:"transactionId"`
		Payment       *CardTokenPayment      `json:"payment,omitempty"`
		Subscription  *CardTokenSubscription `json:"subscription,omitempty"`
		Items         []Item                 `json:"items,omitempty"`
	}

	CreateCardTokenResponse struct {
		ID           string `json:"id"`
		FollowUpLink string `json:"followUpLink"` // Redirect the customer's browser here to enter card details
	}

	PurchaseInput struct {
		Amount        float64 `json:"amount"`
		Currency      string  `json:"currency"` // e.g. "MNT"
		TransactionID string  `json:"transactionId"`
	}

	PurchaseData struct {
		ID          int64             `json:"id"`
		CompletedAt string            `json:"completedAt"`
		Status      TransactionStatus `json:"status"`
		Description string            `json:"description"`
		CardStatus  *string           `json:"cardStatus"` // ACTIVE | INACTIVE, nil when queued
		RespCode    *string           `json:"respCode"`
	}

	PurchaseResponse = Envelope[PurchaseData]

	// RollbackPurchaseResponse has no example body in Bonum's collection; Data is left opaque.
	RollbackPurchaseResponse = Envelope[json.RawMessage]
)
