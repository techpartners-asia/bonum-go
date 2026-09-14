package bonum

import (
	"context"
	"net/http"
)

// PaymentProvider is a payment option that can be shown on the Bonum checkout page.
type PaymentProvider string

const (
	ProviderQPay      PaymentProvider = "QPAY"       // QR based, supported by every bank and fintech app
	ProviderECommerce PaymentProvider = "E_COMMERCE" // Online card payment
	ProviderWeChat    PaymentProvider = "WE_CHAT"    // WeChat wallet
	ProviderSonoShop  PaymentProvider = "SONO_SHOP"  // Buy now, pay later
)

// ExtraType constrains what the customer may type into an Extra on the checkout page.
type ExtraType string

const (
	ExtraText   ExtraType = "TEXT"
	ExtraNumber ExtraType = "NUMBER"
	ExtraPhone  ExtraType = "PHONE"
	ExtraEmail  ExtraType = "EMAIL"
	ExtraAll    ExtraType = "ALL"
)

type (
	PaymentProviderStatus struct {
		Provider PaymentProvider `json:"provider"`
		Enabled  bool            `json:"enabled"`
	}

	// Item is an optional line item rendered on the Bonum checkout / tokenization page.
	Item struct {
		Image  string  `json:"image,omitempty"`
		Title  string  `json:"title"`
		Remark string  `json:"remark,omitempty"`
		Amount float64 `json:"amount"`
		Count  int     `json:"count"`
	}

	// Extra is an additional input field the customer fills in on the checkout page.
	Extra struct {
		Placeholder string    `json:"placeholder"`
		Type        ExtraType `json:"type"`
		Required    bool      `json:"required"`
	}

	CreateInvoiceInput struct {
		Amount        float64           `json:"amount"`
		Callback      string            `json:"callback"`      // Browser is redirected here once payment is done/cancelled/expired
		TransactionID string            `json:"transactionId"` // Merchant's unique id, echoed back in the webhook
		ExpiresIn     int64             `json:"expiresIn"`     // Invoice ttl in seconds
		Providers     []PaymentProvider `json:"providers,omitempty"`
		Items         []Item            `json:"items,omitempty"`
		Extras        []Extra           `json:"extras,omitempty"`
	}

	// Invoice is a checkout request. The outcome arrives as a PaymentEvent on the webhook.
	Invoice struct {
		ID           string `json:"invoiceId"`
		FollowUpLink string `json:"followUpLink"` // Redirect the customer's browser here to pay
	}
)

func (in CreateInvoiceInput) validate() error {
	switch {
	case in.Amount <= 0:
		return invalid("Amount", "must be positive")
	case in.Callback == "":
		return invalid("Callback", "required")
	case in.TransactionID == "":
		return invalid("TransactionID", "required")
	case in.ExpiresIn <= 0:
		return invalid("ExpiresIn", "must be positive seconds")
	}
	return nil
}

// InvoiceService is the Invoice aggregate: hosted checkout.
type InvoiceService struct{ c *Client }

// Providers lists the payment options currently enabled for this Terminal.
func (s *InvoiceService) Providers(ctx context.Context) ([]PaymentProviderStatus, error) {
	out, err := call[[]PaymentProviderStatus](ctx, s.c, http.MethodGet, ecommercePath+"/invoices/payment-providers")
	if err != nil {
		return nil, err
	}
	return *out, nil
}

// Create opens an Invoice. Redirect the customer to FollowUpLink.
func (s *InvoiceService) Create(ctx context.Context, in CreateInvoiceInput) (*Invoice, error) {
	if err := in.validate(); err != nil {
		return nil, err
	}
	return call[Invoice](ctx, s.c, http.MethodPost, ecommercePath+"/invoices", body(in))
}
