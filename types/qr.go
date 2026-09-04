package types

type (
	CreateQrCodeInput struct {
		Amount        float64 `json:"amount"`
		TransactionID string  `json:"transactionId"`
		ExpiresIn     int64   `json:"expiresIn"` // seconds
	}

	// QrLink is a bank-app deeplink that opens the QR invoice directly in that app.
	QrLink struct {
		Name               string `json:"name"`
		Description        string `json:"description"`
		Logo               string `json:"logo"`
		Link               string `json:"link"`
		AppStoreID         string `json:"appStoreId"`
		AndroidPackageName string `json:"androidPackageName"`
	}

	QrCodeData struct {
		InvoiceID string   `json:"invoiceId"`
		QrCode    string   `json:"qrCode"`
		QrImage   string   `json:"qrImage"` // base64 png
		Links     []QrLink `json:"links"`
	}

	CreateQrCodeResponse = Envelope[QrCodeData]

	// InvoiceByQrCodeResponse shape is inferred from CreateQrCodeResponse; the collection has no example.
	InvoiceByQrCodeResponse = Envelope[QrCodeData]

	PayByCardTokenInput struct {
		QrCode        string `json:"qrCode"`
		TransactionID string `json:"transactionId"`
	}

	// PayByCardTokenResponse shape is inferred from PurchaseResponse; the collection has no example.
	PayByCardTokenResponse = Envelope[PurchaseData]
)
