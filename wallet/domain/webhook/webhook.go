// Package webhook is the Wallet Webhook Event aggregate: signed deliveries that report a
// Wallet Payment reaching AUTHORIZED or FAILED.
package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/techpartners-asia/bonum-go/wallet/domain/payment"
)

var (
	ErrMissingSignature  = errors.New("bonum wallet: missing or malformed signature headers")
	ErrTimestampExpired  = errors.New("bonum wallet: webhook timestamp outside replay tolerance")
	ErrSignatureMismatch = errors.New("bonum wallet: webhook signature mismatch")
)

const (
	// SignatureHeader carries "v1=<hex HMAC-SHA256>" on every V2 webhook delivery.
	SignatureHeader = "X-PSP-Signature"
	// TimestampHeader carries the unix seconds at which Bonum sent the delivery.
	TimestampHeader = "X-PSP-Timestamp"

	// ReplayTolerance is how far the timestamp may drift from now before the delivery is rejected.
	ReplayTolerance = 5 * time.Minute

	signaturePrefix = "v1="
)

// Event is the JSON body Bonum POSTs when a payment reaches a final status.
type Event struct {
	WebhookID         string               `json:"webhookId"` // Unique per delivery; use as the idempotency key
	PaymentID         string               `json:"paymentId"`
	OrderID           string               `json:"orderId"`
	EventType         payment.Status       `json:"eventType"` // AUTHORIZED or FAILED
	Status            payment.Status       `json:"status"`    // Mirrors EventType
	Amount            string               `json:"amount"`    // Decimal string in major units
	Currency          string               `json:"currency"`  // e.g. "MNT"
	ProviderReference *string              `json:"providerReference"`
	FailureReason     *string              `json:"failureReason"`
	OccurredAt        string               `json:"occurredAt"` // ISO 8601
	WalletType        payment.WalletType   `json:"walletType"`
	BinCategory       *payment.BinCategory `json:"binCategory"` // nil when the BIN could not be resolved
}

// Sign computes "v1=" + hex(HMAC-SHA256(secret, timestamp + "." + body)) exactly as Bonum does.
func Sign(body []byte, timestamp, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(body)
	return signaturePrefix + hex.EncodeToString(mac.Sum(nil))
}

// Parse is ParseAt against the wall clock.
func Parse(body []byte, signature, timestamp, secret string) (*Event, error) {
	return ParseAt(body, signature, timestamp, secret, time.Now())
}

// ParseAt verifies a delivery against the X-PSP-Signature and X-PSP-Timestamp header values
// as of now, then decodes it. Pass the body bytes exactly as received; re-serialising changes
// the hash. Rejections are ErrMissingSignature, ErrTimestampExpired or ErrSignatureMismatch.
func ParseAt(body []byte, signature, timestamp, secret string, now time.Time) (*Event, error) {
	if err := verify(body, signature, timestamp, secret, now); err != nil {
		return nil, err
	}
	var ev Event
	if err := json.Unmarshal(body, &ev); err != nil {
		return nil, fmt.Errorf("bonum wallet: webhook body: %w", err)
	}
	return &ev, nil
}

func verify(body []byte, signature, timestamp, secret string, now time.Time) error {
	if signature == "" || timestamp == "" {
		return ErrMissingSignature
	}
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return ErrMissingSignature
	}
	drift := now.Sub(time.Unix(ts, 0))
	if drift < 0 {
		drift = -drift
	}
	if drift > ReplayTolerance {
		return ErrTimestampExpired
	}
	if !strings.HasPrefix(signature, signaturePrefix) {
		return ErrSignatureMismatch
	}
	expected := Sign(body, timestamp, secret)
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return ErrSignatureMismatch
	}
	return nil
}
