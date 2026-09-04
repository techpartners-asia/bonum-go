package types

import "encoding/json"

type WebhookType string

const (
	WebhookPayment             WebhookType = "PAYMENT"              // Invoice created via CreateInvoice was paid / failed / expired
	WebhookCardToken           WebhookType = "CARD-TOKEN"           // Card tokenization finished
	WebhookTokenPayment        WebhookType = "TOKEN-PAYMENT"        // A queued Purchase finished asynchronously
	WebhookSubscriptionPayment WebhookType = "SUBSCRIPTION-PAYMENT" // Automatic recurring charge ran
)

type WebhookStatus string

const (
	WebhookSuccess WebhookStatus = "SUCCESS"
	WebhookFailed  WebhookStatus = "FAILED"
)

type (
	// WebhookMessage is the envelope Bonum POSTs to the merchant webhook. Body depends on Type.
	WebhookMessage[T any] struct {
		Type    WebhookType   `json:"type"`
		Status  WebhookStatus `json:"status"`
		Message string        `json:"message"` // Free text, may change at any time - do not rely on it
		Body    T             `json:"body"`
	}

	// WebhookHeader is used to peek at Type/Status before deciding how to decode Body.
	WebhookHeader = WebhookMessage[json.RawMessage]

	// PaymentWebhookBody is a union of the SUCCESS and FAILED payloads; fields absent for one
	// outcome are pointers.
	PaymentWebhookBody struct {
		TransactionID string  `json:"transactionId"`
		Amount        float64 `json:"amount"`
		Currency      string  `json:"currency"`
		TerminalID    string  `json:"terminalId"`

		// SUCCESS only
		InvoiceID     *string          `json:"invoiceId"`
		CompletedAt   *string          `json:"completedAt"`
		PaymentVendor *PaymentProvider `json:"paymentVendor"`
		InitType      *string          `json:"initType"`
		Status        *string          `json:"status"` // e.g. PAID
		RespCode      *string          `json:"respCode"`
		ExtrasInputs  json.RawMessage  `json:"extras-inputs"`
		Extras        json.RawMessage  `json:"extras"`

		// FAILED only
		UpdatedAt     *int64  `json:"updatedAt"`     // epoch millis
		InvoiceStatus *string `json:"invoiceStatus"` // e.g. EXPIRED
	}

	PaymentWebhookMessage = WebhookMessage[PaymentWebhookBody]

	Bank struct {
		ID           int64  `json:"id"`
		Code         string `json:"code"`
		Name         string `json:"name"`
		Icon         string `json:"icon"`
		IBanCode     string `json:"iBanCode"`
		TransferCode string `json:"transferCode"`
	}

	Money struct {
		Amount   float64 `json:"amount"`
		Currency string  `json:"currency"`
	}

	WebhookSubscriptionRef struct {
		SubscriptionID  int64  `json:"subscriptionId"`
		PlanID          int64  `json:"planId"`
		NextBillingDate string `json:"nextBillingDate"`
	}

	CardTokenWebhookBody struct {
		Token         string                   `json:"token"`
		Mask          string                   `json:"mask"`   // e.g. "5150 23** **** 4778"
		Expiry        string                   `json:"expiry"` // e.g. "2026/11"
		Bank          *Bank                    `json:"bank"`
		TransactionID string                   `json:"transactionId"`
		CompletedAt   string                   `json:"completedAt"`
		Amounts       []Money                  `json:"amounts"`
		Subscriptions []WebhookSubscriptionRef `json:"subscriptions"`
	}

	CardTokenWebhookMessage = WebhookMessage[CardTokenWebhookBody]

	TokenPaymentWebhookBody struct {
		TransactionID string `json:"transactionId"`
		CompletedAt   string `json:"completedAt"`
	}

	TokenPaymentWebhookMessage = WebhookMessage[TokenPaymentWebhookBody]

	SubscriptionPaymentWebhookBody struct {
		SubscriptionID int64   `json:"subscriptionId"`
		InvoiceID      int64   `json:"invoiceId"`
		PlanID         int64   `json:"planId"`
		TransactionID  string  `json:"transactionId"`
		CompletedAt    string  `json:"completedAt"`
		Amount         float64 `json:"amount"`
		Currency       string  `json:"currency"`
	}

	SubscriptionPaymentWebhookMessage = WebhookMessage[SubscriptionPaymentWebhookBody]
)
