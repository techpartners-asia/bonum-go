package bonum

import (
	"github.com/techpartners-asia/bonum-go/gateway/application"
	"github.com/techpartners-asia/bonum-go/gateway/domain/qr"
)

// QRService is the QR Invoice aggregate's use cases: QPay-compatible QR and deeplink payments.
type QRService = application.QR

type (
	CreateQRInput = qr.CreateQRInput
	// Deeplink opens the QR Invoice directly in one bank's app.
	Deeplink = qr.Deeplink
	// QRInvoice is a QPay-compatible invoice payable by scanning or via a Deeplink.
	QRInvoice  = qr.QRInvoice
	PayQRInput = qr.PayQRInput
)
