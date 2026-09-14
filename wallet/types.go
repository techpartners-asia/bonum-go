package wallet

import (
	"github.com/techpartners-asia/bonum-go/wallet/domain"
	"github.com/techpartners-asia/bonum-go/wallet/domain/payment"
	"github.com/techpartners-asia/bonum-go/wallet/domain/webhook"
)

// Payment aggregate (wallet/domain/payment).
type (
	Status                = payment.Status
	WalletType            = payment.WalletType
	Currency              = payment.Currency
	BinCategory           = payment.BinCategory
	ApplePaymentHeader    = payment.ApplePaymentHeader
	ApplePaymentData      = payment.ApplePaymentData
	ApplePaymentMethod    = payment.ApplePaymentMethod
	ApplePayToken         = payment.ApplePayToken
	ProcessApplePayInput  = payment.ProcessApplePayInput
	ProcessGooglePayInput = payment.ProcessGooglePayInput
	ProcessResponse       = payment.ProcessResponse
	Payment               = payment.Payment
	AwaitResult           = payment.AwaitResult
)

const (
	StatusPending    = payment.StatusPending
	StatusAuthorized = payment.StatusAuthorized
	StatusFailed     = payment.StatusFailed

	ApplePay  = payment.ApplePay
	GooglePay = payment.GooglePay

	MNT = payment.MNT
	USD = payment.USD
	EUR = payment.EUR
	JPY = payment.JPY

	Domestic      = payment.Domestic
	International = payment.International

	// MaxAwaitTimeout is the server-side cap on the await window.
	MaxAwaitTimeout = payment.MaxAwaitTimeout
)

// Errors (wallet/domain, wallet/domain/webhook).
var (
	ErrInvalidInput      = domain.ErrInvalidInput
	ErrUnauthorized      = domain.ErrUnauthorized
	ErrNotFound          = domain.ErrNotFound
	ErrRateLimited       = domain.ErrRateLimited
	ErrMissingSignature  = webhook.ErrMissingSignature
	ErrTimestampExpired  = webhook.ErrTimestampExpired
	ErrSignatureMismatch = webhook.ErrSignatureMismatch
)

type (
	// APIError is returned when Bonum answers with a non-2xx status.
	APIError = domain.APIError
	// ValidationError is returned before any network call when an input violates an invariant.
	ValidationError = domain.ValidationError
)

// Webhook aggregate (wallet/domain/webhook).
type WebhookEvent = webhook.Event

const (
	SignatureHeader = webhook.SignatureHeader
	TimestampHeader = webhook.TimestampHeader
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
