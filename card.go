package bonum

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

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
		Items         []Item                `json:"items,omitempty"`
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

func (in TokenizeInput) validate() error {
	switch {
	case in.Callback == "":
		return invalid("Callback", "required")
	case in.TransactionID == "":
		return invalid("TransactionID", "required")
	}
	return nil
}

func (in PurchaseInput) validate() error {
	switch {
	case in.Amount <= 0:
		return invalid("Amount", "must be positive")
	case in.Currency == "":
		return invalid("Currency", "required")
	case in.TransactionID == "":
		return invalid("TransactionID", "required")
	}
	return nil
}

// CardService is the Card Token aggregate: tokenization and charges against stored cards.
type CardService struct{ c *Client }

// Tokenize starts a card tokenization flow. Redirect the customer to FollowUpLink.
func (s *CardService) Tokenize(ctx context.Context, in TokenizeInput) (*Tokenization, error) {
	if err := in.validate(); err != nil {
		return nil, err
	}
	return call[Tokenization](ctx, s.c, http.MethodPost, mpayPath+"/cards/tokenize/request", body(in))
}

// Purchase charges a Card Token. Under load Bonum may answer with Status QUEUED; the final
// result then arrives as a TokenPaymentEvent. A bank refusal is returned as *DeclinedError.
func (s *CardService) Purchase(ctx context.Context, cardTok string, in PurchaseInput) (*Purchase, error) {
	if err := in.validate(); err != nil {
		return nil, err
	}
	p, err := callEnveloped[Purchase](ctx, s.c, http.MethodPost, mpayPath+"/transaction/purchase", cardToken(cardTok), body(in))
	if err != nil {
		return nil, asDeclined(err)
	}
	return p, nil
}

// Reverse rolls back a Purchase identified by the merchant TransactionID.
func (s *CardService) Reverse(ctx context.Context, cardTok, transactionID string) error {
	return callAction(ctx, s.c, http.MethodDelete, mpayPath+"/transaction/reverse/{transactionId}",
		cardToken(cardTok), pathParam("transactionId", transactionID))
}

// asDeclined upgrades a 400 whose body is a FAILED Purchase envelope into *DeclinedError.
func asDeclined(err error) error {
	var api *APIError
	if !errors.As(err, &api) || api.StatusCode != 400 {
		return err
	}
	var env envelope[Purchase]
	if json.Unmarshal([]byte(api.Body), &env) != nil || env.Data.Status != PurchaseFailed {
		return err
	}
	return &DeclinedError{APIError: api, Purchase: env.Data}
}
