package bonum

import "github.com/techpartners-asia/bonum-go/internal/gateway/domain/webhook"

// Sentinel errors for webhook verification. Match them with errors.Is.
var (
	ErrBadChecksum  = webhook.ErrBadChecksum  // webhook checksum mismatch
	ErrUnknownEvent = webhook.ErrUnknownEvent // unknown webhook event type
)

// ChecksumHeader carries the HMAC Bonum attaches to every gateway webhook delivery.
const ChecksumHeader = webhook.ChecksumHeader

// EventType identifies which aggregate a webhook delivery is about.
type EventType = webhook.EventType

// Outcome is the SUCCESS / FAILED flag on every delivery.
type Outcome = webhook.Outcome

// Event is one verified webhook delivery. Type-switch on the concrete type:
// *PaymentEvent, *CardTokenEvent, *TokenPaymentEvent, *SubscriptionPaymentEvent.
type Event = webhook.Event

// EventHeader is the envelope common to every delivery.
type EventHeader = webhook.EventHeader

type (
	// PaymentBody is a union of the SUCCESS and FAILED payloads; fields absent for one
	// outcome are pointers.
	PaymentBody  = webhook.PaymentBody
	PaymentEvent = webhook.PaymentEvent

	Bank            = webhook.Bank
	Money           = webhook.Money
	SubscriptionRef = webhook.SubscriptionRef

	CardTokenBody  = webhook.CardTokenBody
	CardTokenEvent = webhook.CardTokenEvent

	TokenPaymentBody  = webhook.TokenPaymentBody
	TokenPaymentEvent = webhook.TokenPaymentEvent

	SubscriptionPaymentBody  = webhook.SubscriptionPaymentBody
	SubscriptionPaymentEvent = webhook.SubscriptionPaymentEvent
)

const (
	EventPayment             = webhook.EventPayment             // An Invoice or QR Invoice was paid / failed / expired
	EventCardToken           = webhook.EventCardToken           // A Tokenization finished
	EventTokenPayment        = webhook.EventTokenPayment        // A QUEUED Purchase finished
	EventSubscriptionPayment = webhook.EventSubscriptionPayment // A recurring charge ran

	OutcomeSuccess = webhook.OutcomeSuccess
	OutcomeFailed  = webhook.OutcomeFailed
)

// Checksum computes hex(HMAC-SHA256(key, body)) exactly as Bonum does for gateway webhooks.
func Checksum(body []byte, checksumKey string) string { return webhook.Checksum(body, checksumKey) }

// ParseWebhook verifies a delivery against the x-checksum-v2 header and decodes it into the
// Event for its type. Pass the body bytes exactly as received; re-serialising changes the hash.
// Returns ErrBadChecksum or ErrUnknownEvent (wrapped) on the two rejection paths.
func ParseWebhook(body []byte, checksumHeader, checksumKey string) (Event, error) {
	return webhook.Parse(body, checksumHeader, checksumKey)
}
