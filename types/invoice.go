package types

// PaymentProvider is a payment option that can be shown on the Bonum checkout page.
type PaymentProvider string

const (
	ProviderQPay      PaymentProvider = "QPAY"       // QR based, supported by every bank and fintech app
	ProviderECommerce PaymentProvider = "E_COMMERCE" // Online card payment
	ProviderWeChat    PaymentProvider = "WE_CHAT"    // WeChat wallet
	ProviderSonoShop  PaymentProvider = "SONO_SHOP"  // Buy now, pay later
)

// ExtraType constrains what the customer may type into an extra input on the checkout page.
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

	CreateInvoiceResponse struct {
		InvoiceID    string `json:"invoiceId"`
		FollowUpLink string `json:"followUpLink"` // Redirect the customer's browser here to pay
	}
)
