// Package payment is the Wallet Payment aggregate: one submission of an Apple Pay or
// Google Pay token and its lifecycle from PENDING to AUTHORIZED or FAILED.
package payment

import (
	"time"

	"github.com/techpartners-asia/bonum-go/wallet/domain"
)

// Status is the lifecycle state of a V2 payment.
type Status string

const (
	StatusPending    Status = "PENDING"    // Queued, awaiting bank response. Do not fulfil.
	StatusAuthorized Status = "AUTHORIZED" // Bank approved; funds reserved. Safe to fulfil.
	StatusFailed     Status = "FAILED"     // Declined or processing error. See FailureReason.
)

// WalletType identifies which wallet produced the token.
type WalletType string

const (
	ApplePay  WalletType = "APPLE_PAY"
	GooglePay WalletType = "GOOGLE_PAY"
)

// Currency is an ISO 4217 alphabetic code accepted by the Google Pay endpoint.
type Currency string

const (
	MNT Currency = "MNT"
	USD Currency = "USD"
	EUR Currency = "EUR"
	JPY Currency = "JPY"
)

// BinCategory classifies the card by its issuing bank's BIN.
type BinCategory string

const (
	Domestic      BinCategory = "DOMESTIC"
	International BinCategory = "INTERNATIONAL"
)

type (
	// ApplePaymentHeader mirrors PKPaymentToken.paymentData.header.
	ApplePaymentHeader struct {
		PublicKeyHash      string `json:"publicKeyHash"`
		EphemeralPublicKey string `json:"ephemeralPublicKey"`
		TransactionID      string `json:"transactionId"`
	}

	// ApplePaymentData mirrors PKPaymentToken.paymentData. Pass it exactly as Apple produced it.
	ApplePaymentData struct {
		Data      string             `json:"data"`      // Base64 encrypted payment data
		Signature string             `json:"signature"` // PKCS #7 detached signature
		Header    ApplePaymentHeader `json:"header"`
		Version   string             `json:"version"` // e.g. "EC_v1"
	}

	// ApplePaymentMethod mirrors PKPaymentToken.paymentMethod.
	ApplePaymentMethod struct {
		DisplayName string `json:"displayName"`
		Network     string `json:"network"`
		Type        string `json:"type"`
	}

	// ApplePayToken is the PKPaymentToken object the app receives from the Apple Pay sheet.
	ApplePayToken struct {
		PaymentData           ApplePaymentData   `json:"paymentData"`
		PaymentMethod         ApplePaymentMethod `json:"paymentMethod"`
		TransactionIdentifier string             `json:"transactionIdentifier"`
	}

	ProcessApplePayInput struct {
		OrderID  string        `json:"order_id"`            // Required, unique per merchant, max 128 chars
		Amount   float64       `json:"amount,omitempty"`    // Optional, major units, min 0.01; overrides the token amount
		BranchID string        `json:"branch_id,omitempty"` // Optional, 1-64 chars
		Token    ApplePayToken `json:"token"`
	}

	ProcessGooglePayInput struct {
		OrderID      string   `json:"order_id"`
		Token        string   `json:"token"`         // paymentMethodData.tokenizationData.token, unmodified
		CurrencyCode Currency `json:"currency_code"` // Required
		Amount       float64  `json:"amount,omitempty"`
		BranchID     string   `json:"branch_id,omitempty"`
	}

	// ProcessResponse is returned by both process endpoints. Status is always PENDING here.
	ProcessResponse struct {
		PaymentID  string `json:"paymentId"`
		OrderID    string `json:"orderId"`
		Status     Status `json:"status"`
		AcceptedAt string `json:"acceptedAt"` // ISO 8601
		StatusURL  string `json:"statusUrl"`  // Absolute URL
		AwaitURL   string `json:"awaitUrl"`   // Absolute URL; see AwaitPayment
	}

	// Payment is the full record returned by GetPayment and LookupByOrderID.
	Payment struct {
		PaymentID         string     `json:"paymentId"`
		OrderID           string     `json:"orderId"`
		Amount            string     `json:"amount"`   // Decimal string in major units, e.g. "150.50"
		Currency          string     `json:"currency"` // Docs show both ISO numeric ("496") and alpha ("MNT")
		WalletType        WalletType `json:"walletType"`
		Status            Status     `json:"status"`
		ProviderReference *string    `json:"providerReference"` // Set when AUTHORIZED
		FailureReason     *string    `json:"failureReason"`     // Set when FAILED
		CreatedAt         string     `json:"createdAt"`
		UpdatedAt         string     `json:"updatedAt"`
	}

	// AwaitResult is the trimmed response of the blocking await endpoint.
	AwaitResult struct {
		PaymentID     string  `json:"paymentId"`
		Status        Status  `json:"status"` // AUTHORIZED, FAILED, or PENDING when TimedOut
		FailureReason *string `json:"failureReason"`
		TimedOut      bool    `json:"timedOut"` // True when the wait window expired; the payment keeps processing
	}
)

// MaxAwaitTimeout is the server-side cap on the await window, chosen by Bonum to leave
// a buffer before Apple Pay's 30-second hard limit.
const MaxAwaitTimeout = 28 * time.Second

// --- invariants, checked before any network call ------------------------------------------

const (
	maxOrderIDLen  = 128
	maxBranchIDLen = 64
	minAmount      = 0.01
)

func validateCommon(orderID string, amount float64, branchID string) error {
	switch {
	case orderID == "":
		return domain.Invalid("OrderID", "required")
	case len(orderID) > maxOrderIDLen:
		return domain.Invalid("OrderID", "at most 128 characters")
	case amount != 0 && amount < minAmount:
		return domain.Invalid("Amount", "minimum 0.01")
	case amount != 0 && !hasAtMostTwoDecimals(amount):
		return domain.Invalid("Amount", "at most 2 decimal places")
	case len(branchID) > maxBranchIDLen:
		return domain.Invalid("BranchID", "at most 64 characters")
	}
	return nil
}

func hasAtMostTwoDecimals(v float64) bool {
	cents := v * 100
	return cents-float64(int64(cents+0.5)) < 1e-6 && cents-float64(int64(cents+0.5)) > -1e-6
}

func (in ProcessApplePayInput) Validate() error {
	if err := validateCommon(in.OrderID, in.Amount, in.BranchID); err != nil {
		return err
	}
	if in.Token.PaymentData.Data == "" || in.Token.TransactionIdentifier == "" {
		return domain.Invalid("Token", "must be the PKPaymentToken as received from Apple Pay")
	}
	return nil
}

func (in ProcessGooglePayInput) Validate() error {
	if err := validateCommon(in.OrderID, in.Amount, in.BranchID); err != nil {
		return err
	}
	if in.Token == "" {
		return domain.Invalid("Token", "required")
	}
	switch in.CurrencyCode {
	case MNT, USD, EUR, JPY:
		return nil
	}
	return domain.Invalid("CurrencyCode", "must be MNT, USD, EUR or JPY")
}
