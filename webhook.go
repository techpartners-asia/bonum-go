package bonum

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/techpartners-asia/bonum-go/types"
)

// ChecksumHeader carries the HMAC Bonum attaches to every webhook delivery.
const ChecksumHeader = "x-checksum-v2"

// Checksum computes hex(HMAC-SHA256(key, body)) exactly as Bonum does for webhooks.
func Checksum(body []byte, merchantChecksumKey string) string {
	mac := hmac.New(sha256.New, []byte(merchantChecksumKey))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyWebhook checks the raw request body against the x-checksum-v2 header value.
// Pass the body bytes exactly as received - re-serialising the JSON can change the hash.
func VerifyWebhook(body []byte, checksumHeader, merchantChecksumKey string) bool {
	expected := Checksum(body, merchantChecksumKey)
	return hmac.Equal([]byte(expected), []byte(checksumHeader))
}

// PeekWebhook decodes only the envelope so the caller can dispatch on Type.
func PeekWebhook(body []byte) (*types.WebhookHeader, error) {
	var h types.WebhookHeader
	if err := json.Unmarshal(body, &h); err != nil {
		return nil, err
	}
	return &h, nil
}

func ParsePaymentWebhook(body []byte) (*types.PaymentWebhookMessage, error) {
	return parseWebhook[types.PaymentWebhookBody](body, types.WebhookPayment)
}

func ParseCardTokenWebhook(body []byte) (*types.CardTokenWebhookMessage, error) {
	return parseWebhook[types.CardTokenWebhookBody](body, types.WebhookCardToken)
}

func ParseTokenPaymentWebhook(body []byte) (*types.TokenPaymentWebhookMessage, error) {
	return parseWebhook[types.TokenPaymentWebhookBody](body, types.WebhookTokenPayment)
}

func ParseSubscriptionPaymentWebhook(body []byte) (*types.SubscriptionPaymentWebhookMessage, error) {
	return parseWebhook[types.SubscriptionPaymentWebhookBody](body, types.WebhookSubscriptionPayment)
}

func parseWebhook[T any](body []byte, want types.WebhookType) (*types.WebhookMessage[T], error) {
	var m types.WebhookMessage[T]
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, err
	}
	if m.Type != want {
		return nil, fmt.Errorf("bonum: webhook type %q, expected %q", m.Type, want)
	}
	return &m, nil
}
