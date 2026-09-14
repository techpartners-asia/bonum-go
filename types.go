package bonum

import (
	"github.com/techpartners-asia/bonum-go/gateway/application"
	"github.com/techpartners-asia/bonum-go/gateway/domain/access"
	"github.com/techpartners-asia/bonum-go/gateway/domain/card"
	"github.com/techpartners-asia/bonum-go/gateway/domain/checkout"
	"github.com/techpartners-asia/bonum-go/gateway/domain/qr"
	"github.com/techpartners-asia/bonum-go/gateway/domain/subscription"
	"github.com/techpartners-asia/bonum-go/gateway/domain/webhook"
)

// Services (gateway/application).
type (
	InvoiceService      = application.Invoices
	CardService         = application.Cards
	SubscriptionService = application.Subscriptions
	QRService           = application.QR
	SandboxService      = application.Sandbox
)

// Access (gateway/domain/access).
type TokenPair = access.TokenPair

// Checkout (gateway/domain/checkout).
type (
	PaymentProvider       = checkout.PaymentProvider
	ExtraType             = checkout.ExtraType
	PaymentProviderStatus = checkout.PaymentProviderStatus
	Item                  = checkout.Item
	Extra                 = checkout.Extra
	CreateInvoiceInput    = checkout.CreateInvoiceInput
	Invoice               = checkout.Invoice
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

// Cards (gateway/domain/card).
type (
	PurchaseStatus       = card.PurchaseStatus
	TokenizePayment      = card.TokenizePayment
	TokenizeSubscription = card.TokenizeSubscription
	TokenizeInput        = card.TokenizeInput
	Tokenization         = card.Tokenization
	PurchaseInput        = card.PurchaseInput
	Purchase             = card.Purchase
)

const (
	PurchaseSuccess = card.PurchaseSuccess
	PurchaseFailed  = card.PurchaseFailed
	PurchaseQueued  = card.PurchaseQueued
)

// Subscriptions (gateway/domain/subscription).
type (
	RecurringType   = subscription.RecurringType
	PaymentPlan     = subscription.PaymentPlan
	SubscribeInput  = subscription.SubscribeInput
	Subscription    = subscription.Subscription
	ChangeCardInput = subscription.ChangeCardInput
)

const (
	RecurringWeekly  = subscription.RecurringWeekly
	RecurringMonthly = subscription.RecurringMonthly
	RecurringYearly  = subscription.RecurringYearly
)

// QR (gateway/domain/qr).
type (
	CreateQRInput = qr.CreateQRInput
	Deeplink      = qr.Deeplink
	QRInvoice     = qr.QRInvoice
	PayQRInput    = qr.PayQRInput
)

// Webhook (gateway/domain/webhook).
type (
	EventType                = webhook.EventType
	Outcome                  = webhook.Outcome
	Event                    = webhook.Event
	EventHeader              = webhook.EventHeader
	PaymentBody              = webhook.PaymentBody
	PaymentEvent             = webhook.PaymentEvent
	Bank                     = webhook.Bank
	Money                    = webhook.Money
	SubscriptionRef          = webhook.SubscriptionRef
	CardTokenBody            = webhook.CardTokenBody
	CardTokenEvent           = webhook.CardTokenEvent
	TokenPaymentBody         = webhook.TokenPaymentBody
	TokenPaymentEvent        = webhook.TokenPaymentEvent
	SubscriptionPaymentBody  = webhook.SubscriptionPaymentBody
	SubscriptionPaymentEvent = webhook.SubscriptionPaymentEvent
)

const (
	EventPayment             = webhook.EventPayment
	EventCardToken           = webhook.EventCardToken
	EventTokenPayment        = webhook.EventTokenPayment
	EventSubscriptionPayment = webhook.EventSubscriptionPayment

	OutcomeSuccess = webhook.OutcomeSuccess
	OutcomeFailed  = webhook.OutcomeFailed
)
