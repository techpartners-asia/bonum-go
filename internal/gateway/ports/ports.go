// Package ports declares the outbound capabilities the Gateway application layer depends on:
// one interface per aggregate, all implemented by adapters/httpapi. Method names are unique
// across the interfaces so a single adapter can satisfy every port.
package ports

import (
	"context"
	"encoding/json"

	"github.com/techpartners-asia/bonum-go/internal/gateway/domain/access"
	"github.com/techpartners-asia/bonum-go/internal/gateway/domain/card"
	"github.com/techpartners-asia/bonum-go/internal/gateway/domain/checkout"
	"github.com/techpartners-asia/bonum-go/internal/gateway/domain/qr"
	"github.com/techpartners-asia/bonum-go/internal/gateway/domain/subscription"
)

// AccessAPI exchanges the Terminal credential for bearer tokens.
type AccessAPI interface {
	CreateToken(ctx context.Context) (*access.TokenPair, error)
	RefreshToken(ctx context.Context) (*access.TokenPair, error)
}

// CheckoutAPI opens hosted checkout Invoices.
type CheckoutAPI interface {
	Providers(ctx context.Context) ([]checkout.PaymentProviderStatus, error)
	CreateInvoice(ctx context.Context, in checkout.CreateInvoiceInput) (*checkout.Invoice, error)
}

// CardAPI tokenizes cards and charges Card Tokens. Purchase returns *card.DeclinedError
// when the bank refuses.
type CardAPI interface {
	Tokenize(ctx context.Context, in card.TokenizeInput) (*card.Tokenization, error)
	Purchase(ctx context.Context, cardToken string, in card.PurchaseInput) (*card.Purchase, error)
	Reverse(ctx context.Context, cardToken, transactionID string) error
}

// SubscriptionAPI manages Card Tokens enrolled in Payment Plans.
type SubscriptionAPI interface {
	Plans(ctx context.Context) ([]subscription.PaymentPlan, error)
	Subscribe(ctx context.Context, cardToken string, in subscription.SubscribeInput) (*subscription.Subscription, error)
	ListSubscriptions(ctx context.Context, cardToken string) ([]subscription.Subscription, error)
	ChangeCardByTokenizing(ctx context.Context, id int64, in subscription.ChangeCardInput) (*card.Tokenization, error)
	ChangeCard(ctx context.Context, id int64, cardToken string) (*subscription.Subscription, error)
	Unsubscribe(ctx context.Context, id, planID int64) error
	DeleteSubscription(ctx context.Context, id, planID int64) error
}

// QRAPI creates and settles QR Invoices.
type QRAPI interface {
	CreateQR(ctx context.Context, in qr.CreateQRInput) (*qr.QRInvoice, error)
	LookupQR(ctx context.Context, qrCode string) (*qr.QRInvoice, error)
	PayQRWithCard(ctx context.Context, cardToken string, in qr.PayQRInput) (*card.Purchase, error)
}

// SandboxAPI groups helpers Bonum only permits outside production.
type SandboxAPI interface {
	InvoiceStatus(ctx context.Context, invoiceID string) (json.RawMessage, error)
	MarkInvoicePaid(ctx context.Context, invoiceID string) error
	RunSubscriptionBilling(ctx context.Context, id int64) error
}
