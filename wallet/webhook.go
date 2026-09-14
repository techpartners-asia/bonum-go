package wallet

import "github.com/techpartners-asia/bonum-go/internal/wallet/domain/webhook"

// Sentinel errors for webhook verification. Match them with errors.Is.
var (
	ErrMissingSignature  = webhook.ErrMissingSignature
	ErrTimestampExpired  = webhook.ErrTimestampExpired
	ErrSignatureMismatch = webhook.ErrSignatureMismatch
)

// WebhookEvent is the JSON body Bonum POSTs when a payment reaches a final status.
type WebhookEvent = webhook.Event

const (
	// SignatureHeader carries "v1=<hex HMAC-SHA256>" on every V2 webhook delivery.
	SignatureHeader = webhook.SignatureHeader
	// TimestampHeader carries the unix seconds at which Bonum sent the delivery.
	TimestampHeader = webhook.TimestampHeader
	// ReplayTolerance is how far the timestamp may drift from now before the delivery is rejected.
	ReplayTolerance = webhook.ReplayTolerance
)

// Sign computes "v1=" + hex(HMAC-SHA256(secret, timestamp + "." + body)) exactly as Bonum does.
func Sign(body []byte, timestamp, secret string) string { return webhook.Sign(body, timestamp, secret) }

// ParseWebhook verifies a delivery against the X-PSP-Signature and X-PSP-Timestamp header
// values and decodes it. Pass the body bytes exactly as received; re-serialising changes the
// hash. Rejections are ErrMissingSignature, ErrTimestampExpired or ErrSignatureMismatch.
func ParseWebhook(body []byte, signature, timestamp, secret string) (*WebhookEvent, error) {
	return webhook.Parse(body, signature, timestamp, secret)
}
