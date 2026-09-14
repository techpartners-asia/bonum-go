// Package qr is the QR Invoice aggregate: QPay-compatible QR and deeplink payments.
package qr

import "github.com/techpartners-asia/bonum-go/internal/gateway/domain"

type (
	CreateQRInput struct {
		Amount        float64 `json:"amount"`
		TransactionID string  `json:"transactionId"`
		ExpiresIn     int64   `json:"expiresIn"` // seconds
	}

	// Deeplink opens the QR Invoice directly in one bank's app.
	Deeplink struct {
		Name               string `json:"name"`
		Description        string `json:"description"`
		Logo               string `json:"logo"`
		Link               string `json:"link"`
		AppStoreID         string `json:"appStoreId"`
		AndroidPackageName string `json:"androidPackageName"`
	}

	// QRInvoice is a QPay-compatible invoice payable by scanning or via a Deeplink.
	QRInvoice struct {
		InvoiceID string     `json:"invoiceId"`
		QrCode    string     `json:"qrCode"`
		QrImage   string     `json:"qrImage"` // base64 png
		Links     []Deeplink `json:"links"`
	}

	PayQRInput struct {
		QrCode        string `json:"qrCode"`
		TransactionID string `json:"transactionId"`
	}
)

// Validate enforces the QR Invoice invariants before any network call.
func (in CreateQRInput) Validate() error {
	switch {
	case in.Amount <= 0:
		return domain.Invalid("Amount", "must be positive")
	case in.TransactionID == "":
		return domain.Invalid("TransactionID", "required")
	case in.ExpiresIn <= 0:
		return domain.Invalid("ExpiresIn", "must be positive seconds")
	}
	return nil
}

// Validate enforces the QR payment invariants before any network call.
func (in PayQRInput) Validate() error {
	switch {
	case in.QrCode == "":
		return domain.Invalid("QrCode", "required")
	case in.TransactionID == "":
		return domain.Invalid("TransactionID", "required")
	}
	return nil
}
