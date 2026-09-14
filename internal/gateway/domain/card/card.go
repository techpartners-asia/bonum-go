// Package card is the Card Token aggregate: tokenization and purchases against stored cards.
package card

import (
	"errors"

	"github.com/techpartners-asia/bonum-go/internal/gateway/domain"
	"github.com/techpartners-asia/bonum-go/internal/gateway/domain/checkout"
)

// ErrDeclined marks a Purchase the bank refused. Match with errors.Is; the *DeclinedError
// carrying Bonum's Purchase record is available through errors.As.
var ErrDeclined = errors.New("bonum: purchase declined")

// PurchaseStatus is the outcome of a Purchase.
type PurchaseStatus string

const (
	PurchaseSuccess PurchaseStatus = "SUCCESS"
	PurchaseFailed  PurchaseStatus = "FAILED"
	PurchaseQueued  PurchaseStatus = "QUEUED" // Result arrives later as a TokenPaymentEvent
)

type (
	// TokenizePayment charges the card while it is being tokenized. Defaults to 0.01 MNT if omitted.
	TokenizePayment struct {
		Amount float64 `json:"amount"`
	}

	// TokenizeSubscription enrols the new Card Token in a Payment Plan in the same flow.
	TokenizeSubscription struct {
		PlanID     int64  `json:"planId"`
		CycleValue string `json:"cycleValue"` // 1-7 weekly, 1-31 monthly, 1-366 yearly
		Cycles     *int64 `json:"cycles,omitempty"`
		PayNow     bool   `json:"payNow"`
		CustEmail  string `json:"custEmail,omitempty"`
	}

	TokenizeInput struct {
		Callback      string                `json:"callback"`
		TransactionID string                `json:"transactionId"`
		Payment       *TokenizePayment      `json:"payment,omitempty"`
		Subscription  *TokenizeSubscription `json:"subscription,omitempty"`
		Items         []checkout.Item       `json:"items,omitempty"`
	}

	// Tokenization is a hosted card-entry session. The Card Token arrives as a CardTokenEvent.
	Tokenization struct {
		ID           string `json:"id"`
		FollowUpLink string `json:"followUpLink"` // Redirect the customer's browser here to enter card details
	}

	PurchaseInput struct {
		Amount        float64 `json:"amount"`
		Currency      string  `json:"currency"` // e.g. "MNT"
		TransactionID string  `json:"transactionId"`
	}

	// Purchase is a charge against a Card Token.
	Purchase struct {
		ID          int64          `json:"id"`
		CompletedAt string         `json:"completedAt"`
		Status      PurchaseStatus `json:"status"`
		Description string         `json:"description"`
		CardStatus  *string        `json:"cardStatus"` // ACTIVE | INACTIVE, nil when queued
		RespCode    *string        `json:"respCode"`
	}
)

// Validate enforces the Tokenization invariants before any network call.
func (in TokenizeInput) Validate() error {
	switch {
	case in.Callback == "":
		return domain.Invalid("Callback", "required")
	case in.TransactionID == "":
		return domain.Invalid("TransactionID", "required")
	}
	return nil
}

// Validate enforces the Purchase invariants before any network call.
func (in PurchaseInput) Validate() error {
	switch {
	case in.Amount <= 0:
		return domain.Invalid("Amount", "must be positive")
	case in.Currency == "":
		return domain.Invalid("Currency", "required")
	case in.TransactionID == "":
		return domain.Invalid("TransactionID", "required")
	}
	return nil
}

// DeclinedError is an APIError whose body carried a FAILED Purchase: the bank refused the
// card. Purchase holds Bonum's record of the attempt (id, respCode, cardStatus).
type DeclinedError struct {
	*domain.APIError
	Purchase Purchase
}

func (e *DeclinedError) Is(target error) bool { return target == ErrDeclined || e.APIError.Is(target) }
func (e *DeclinedError) Unwrap() error        { return e.APIError }
