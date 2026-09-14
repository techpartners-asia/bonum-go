// Package webhook is the Event aggregate: verified gateway webhook deliveries.
package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/techpartners-asia/bonum-go/gateway/domain/checkout"
)

var (
	ErrBadChecksum  = errors.New("bonum: webhook checksum mismatch")
	ErrUnknownEvent = errors.New("bonum: unknown webhook event type")
)

// ChecksumHeader carries the HMAC Bonum attaches to every gateway webhook delivery.
const ChecksumHeader = "x-checksum-v2"

// EventType identifies which aggregate a webhook delivery is about.
type EventType string

const (
	EventPayment             EventType = "PAYMENT"              // An Invoice or QR Invoice was paid / failed / expired
	EventCardToken           EventType = "CARD-TOKEN"           // A Tokenization finished
	EventTokenPayment        EventType = "TOKEN-PAYMENT"        // A QUEUED Purchase finished
	EventSubscriptionPayment EventType = "SUBSCRIPTION-PAYMENT" // A recurring charge ran
)

// Outcome is the SUCCESS / FAILED flag on every delivery.
type Outcome string

const (
	OutcomeSuccess Outcome = "SUCCESS"
	OutcomeFailed  Outcome = "FAILED"
)

// Event is one verified webhook delivery. Type-switch on the concrete type:
// *PaymentEvent, *CardTokenEvent, *TokenPaymentEvent, *SubscriptionPaymentEvent.
type Event interface {
	Header() EventHeader
}

// EventHeader is the envelope common to every delivery.
type EventHeader struct {
	Type    EventType `json:"type"`
	Outcome Outcome   `json:"status"`
	Message string    `json:"message"` // Free text, may change at any time - do not rely on it
}

func (h EventHeader) Header() EventHeader { return h }

type (
	// PaymentBody is a union of the SUCCESS and FAILED payloads; fields absent for one
	// outcome are pointers.
	PaymentBody struct {
		TransactionID string  `json:"transactionId"`
		Amount        float64 `json:"amount"`
		Currency      string  `json:"currency"`
		TerminalID    string  `json:"terminalId"`

		// SUCCESS only
		InvoiceID     *string                   `json:"invoiceId"`
		CompletedAt   *string                   `json:"completedAt"`
		PaymentVendor *checkout.PaymentProvider `json:"paymentVendor"`
		InitType      *string                   `json:"initType"`
		Status        *string                   `json:"status"` // e.g. PAID
		RespCode      *string                   `json:"respCode"`
		ExtrasInputs  json.RawMessage           `json:"extras-inputs"`
		Extras        json.RawMessage           `json:"extras"`

		// FAILED only
		UpdatedAt     *int64  `json:"updatedAt"`     // epoch millis
		InvoiceStatus *string `json:"invoiceStatus"` // e.g. EXPIRED
	}

	PaymentEvent struct {
		EventHeader
		Body PaymentBody `json:"body"`
	}

	Bank struct {
		ID           int64  `json:"id"`
		Code         string `json:"code"`
		Name         string `json:"name"`
		Icon         string `json:"icon"`
		IBanCode     string `json:"iBanCode"`
		TransferCode string `json:"transferCode"`
	}

	Money struct {
		Amount   float64 `json:"amount"`
		Currency string  `json:"currency"`
	}

	SubscriptionRef struct {
		SubscriptionID  int64  `json:"subscriptionId"`
		PlanID          int64  `json:"planId"`
		NextBillingDate string `json:"nextBillingDate"`
	}

	CardTokenBody struct {
		Token         string            `json:"token"`
		Mask          string            `json:"mask"`   // e.g. "5150 23** **** 4778"
		Expiry        string            `json:"expiry"` // e.g. "2026/11"
		Bank          *Bank             `json:"bank"`
		TransactionID string            `json:"transactionId"`
		CompletedAt   string            `json:"completedAt"`
		Amounts       []Money           `json:"amounts"`
		Subscriptions []SubscriptionRef `json:"subscriptions"`
	}

	CardTokenEvent struct {
		EventHeader
		Body CardTokenBody `json:"body"`
	}

	TokenPaymentBody struct {
		TransactionID string `json:"transactionId"`
		CompletedAt   string `json:"completedAt"`
	}

	TokenPaymentEvent struct {
		EventHeader
		Body TokenPaymentBody `json:"body"`
	}

	SubscriptionPaymentBody struct {
		SubscriptionID int64   `json:"subscriptionId"`
		InvoiceID      int64   `json:"invoiceId"`
		PlanID         int64   `json:"planId"`
		TransactionID  string  `json:"transactionId"`
		CompletedAt    string  `json:"completedAt"`
		Amount         float64 `json:"amount"`
		Currency       string  `json:"currency"`
	}

	SubscriptionPaymentEvent struct {
		EventHeader
		Body SubscriptionPaymentBody `json:"body"`
	}
)

// Checksum computes hex(HMAC-SHA256(key, body)) exactly as Bonum does for gateway webhooks.
func Checksum(body []byte, checksumKey string) string {
	mac := hmac.New(sha256.New, []byte(checksumKey))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// Parse verifies a delivery against the x-checksum-v2 header and decodes it into the Event
// for its type. Pass the body bytes exactly as received; re-serialising changes the hash.
func Parse(body []byte, checksumHeader, checksumKey string) (Event, error) {
	if !hmac.Equal([]byte(Checksum(body, checksumKey)), []byte(checksumHeader)) {
		return nil, ErrBadChecksum
	}
	var h EventHeader
	if err := json.Unmarshal(body, &h); err != nil {
		return nil, fmt.Errorf("bonum: webhook body: %w", err)
	}
	switch h.Type {
	case EventPayment:
		return decodeEvent[PaymentEvent](body)
	case EventCardToken:
		return decodeEvent[CardTokenEvent](body)
	case EventTokenPayment:
		return decodeEvent[TokenPaymentEvent](body)
	case EventSubscriptionPayment:
		return decodeEvent[SubscriptionPaymentEvent](body)
	}
	return nil, fmt.Errorf("%w: %q", ErrUnknownEvent, h.Type)
}

func decodeEvent[T any, PT interface {
	*T
	Event
}](body []byte) (Event, error) {
	var ev T
	if err := json.Unmarshal(body, &ev); err != nil {
		return nil, fmt.Errorf("bonum: webhook body: %w", err)
	}
	return PT(&ev), nil
}
