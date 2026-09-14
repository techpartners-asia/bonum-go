package bonum

import (
	"context"
	"net/http"
)

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

func (in CreateQRInput) validate() error {
	switch {
	case in.Amount <= 0:
		return invalid("Amount", "must be positive")
	case in.TransactionID == "":
		return invalid("TransactionID", "required")
	case in.ExpiresIn <= 0:
		return invalid("ExpiresIn", "must be positive seconds")
	}
	return nil
}

func (in PayQRInput) validate() error {
	switch {
	case in.QrCode == "":
		return invalid("QrCode", "required")
	case in.TransactionID == "":
		return invalid("TransactionID", "required")
	}
	return nil
}

// QRService is the QR Invoice aggregate: QPay-compatible QR and deeplink payments.
type QRService struct{ c *Client }

// Create opens a QR Invoice. Render QrImage or offer Links; the outcome arrives as a PaymentEvent.
func (s *QRService) Create(ctx context.Context, in CreateQRInput) (*QRInvoice, error) {
	if err := in.validate(); err != nil {
		return nil, err
	}
	return callEnveloped[QRInvoice](ctx, s.c, http.MethodPost, mpayPath+"/transaction/qr/create", body(in))
}

// Lookup returns the QR Invoice behind a scanned QPay QR string.
func (s *QRService) Lookup(ctx context.Context, qrCode string) (*QRInvoice, error) {
	if qrCode == "" {
		return nil, invalid("qrCode", "required")
	}
	return callEnveloped[QRInvoice](ctx, s.c, http.MethodPost, mpayPath+"/transaction/qr", body(map[string]string{"qrCode": qrCode}))
}

// PayWithCard settles a QR Invoice with a stored Card Token.
func (s *QRService) PayWithCard(ctx context.Context, cardTok string, in PayQRInput) (*Purchase, error) {
	if err := in.validate(); err != nil {
		return nil, err
	}
	return callEnveloped[Purchase](ctx, s.c, http.MethodPut, mpayPath+"/transaction/qr/pay", cardToken(cardTok), body(in))
}
