package wallet

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
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

// WebhookEvent is the JSON body Bonum POSTs when a payment reaches a final status.
type WebhookEvent struct {
	WebhookID         string       `json:"webhookId"` // Unique per delivery; use as the idempotency key
	PaymentID         string       `json:"paymentId"`
	OrderID           string       `json:"orderId"`
	EventType         Status       `json:"eventType"` // AUTHORIZED or FAILED
	Status            Status       `json:"status"`    // Mirrors EventType
	Amount            string       `json:"amount"`    // Decimal string in major units
	Currency          string       `json:"currency"`  // e.g. "MNT"
	ProviderReference *string      `json:"providerReference"`
	FailureReason     *string      `json:"failureReason"`
	OccurredAt        string       `json:"occurredAt"` // ISO 8601
	WalletType        WalletType   `json:"walletType"`
	BinCategory       *BinCategory `json:"binCategory"` // nil when the BIN could not be resolved
}

// Sign computes "v1=" + hex(HMAC-SHA256(secret, timestamp + "." + body)) exactly as Bonum does.
func Sign(body []byte, timestamp, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(body)
	return signaturePrefix + hex.EncodeToString(mac.Sum(nil))
}

// ParseWebhook verifies a delivery against the X-PSP-Signature and X-PSP-Timestamp header
// values and decodes it. Pass the body bytes exactly as received; re-serialising changes the
// hash. Rejections are ErrMissingSignature, ErrTimestampExpired or ErrSignatureMismatch.
func ParseWebhook(body []byte, signature, timestamp, secret string) (*WebhookEvent, error) {
	if err := verify(body, signature, timestamp, secret, time.Now()); err != nil {
		return nil, err
	}
	var ev WebhookEvent
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
