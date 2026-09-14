package wallet

import "github.com/techpartners-asia/bonum-go/wallet/domain/payment"

// Status is the lifecycle state of a V2 payment.
type Status = payment.Status

// WalletType identifies which wallet produced the token.
type WalletType = payment.WalletType

// Currency is an ISO 4217 alphabetic code accepted by the Google Pay endpoint.
type Currency = payment.Currency

// BinCategory classifies the card by its issuing bank's BIN.
type BinCategory = payment.BinCategory

type (
	// ApplePaymentHeader mirrors PKPaymentToken.paymentData.header.
	ApplePaymentHeader = payment.ApplePaymentHeader
	// ApplePaymentData mirrors PKPaymentToken.paymentData.
	ApplePaymentData = payment.ApplePaymentData
	// ApplePaymentMethod mirrors PKPaymentToken.paymentMethod.
	ApplePaymentMethod = payment.ApplePaymentMethod
	// ApplePayToken is the PKPaymentToken object the app receives from the Apple Pay sheet.
	ApplePayToken         = payment.ApplePayToken
	ProcessApplePayInput  = payment.ProcessApplePayInput
	ProcessGooglePayInput = payment.ProcessGooglePayInput
	// ProcessResponse is returned by both process endpoints. Status is always PENDING here.
	ProcessResponse = payment.ProcessResponse
	// Payment is the full record returned by GetPayment and LookupByOrderID.
	Payment = payment.Payment
	// AwaitResult is the trimmed response of the blocking await endpoint.
	AwaitResult = payment.AwaitResult
)

const (
	StatusPending    = payment.StatusPending    // Queued, awaiting bank response. Do not fulfil.
	StatusAuthorized = payment.StatusAuthorized // Bank approved; funds reserved. Safe to fulfil.
	StatusFailed     = payment.StatusFailed     // Declined or processing error. See FailureReason.

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
