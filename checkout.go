package bonum

import (
	"github.com/techpartners-asia/bonum-go/internal/gateway/application"
	"github.com/techpartners-asia/bonum-go/internal/gateway/domain/checkout"
)

// InvoiceService is the Invoice aggregate's use cases: hosted checkout.
type InvoiceService = application.Invoices

// PaymentProvider is a payment option that can be shown on the Bonum checkout page.
type PaymentProvider = checkout.PaymentProvider

// ExtraType constrains what the customer may type into an Extra on the checkout page.
type ExtraType = checkout.ExtraType

type (
	PaymentProviderStatus = checkout.PaymentProviderStatus
	// Item is an optional line item rendered on the Bonum checkout / tokenization page.
	Item = checkout.Item
	// Extra is an additional input field the customer fills in on the checkout page.
	Extra              = checkout.Extra
	CreateInvoiceInput = checkout.CreateInvoiceInput
	// Invoice is a checkout request. The outcome arrives as a PaymentEvent on the webhook.
	Invoice = checkout.Invoice
)

const (
	ProviderQPay      = checkout.ProviderQPay
	ProviderECommerce = checkout.ProviderECommerce
	ProviderWeChat    = checkout.ProviderWeChat
	ProviderSonoShop  = checkout.ProviderSonoShop

	ExtraText   = checkout.ExtraText
	ExtraNumber = checkout.ExtraNumber
	ExtraPhone  = checkout.ExtraPhone
	ExtraEmail  = checkout.ExtraEmail
	ExtraAll    = checkout.ExtraAll
)
