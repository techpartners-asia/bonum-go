package application

import (
	"context"

	qrcmd "github.com/techpartners-asia/bonum-go/gateway/application/commands/qr"
	qrqry "github.com/techpartners-asia/bonum-go/gateway/application/queries/qr"
	"github.com/techpartners-asia/bonum-go/gateway/domain/card"
	"github.com/techpartners-asia/bonum-go/gateway/domain/qr"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// QR is the QR Invoice aggregate's use cases: QPay-compatible QR and deeplink payments.
type QR struct {
	create      *qrcmd.CreateQRHandler
	lookup      *qrqry.LookupQRHandler
	payWithCard *qrcmd.PayQRHandler
}

func NewQR(api ports.QRAPI) *QR {
	return &QR{
		create:      qrcmd.NewCreateQRHandler(api),
		lookup:      qrqry.NewLookupQRHandler(api),
		payWithCard: qrcmd.NewPayQRHandler(api),
	}
}

// Create opens a QR Invoice. Render QrImage or offer Links; the outcome arrives as a PaymentEvent.
func (s *QR) Create(ctx context.Context, in qr.CreateQRInput) (*qr.QRInvoice, error) {
	return s.create.Handle(ctx, in)
}

// Lookup returns the QR Invoice behind a scanned QPay QR string.
func (s *QR) Lookup(ctx context.Context, qrCode string) (*qr.QRInvoice, error) {
	return s.lookup.Handle(ctx, qrqry.LookupQRQuery{QrCode: qrCode})
}

// PayWithCard settles a QR Invoice with a stored Card Token.
func (s *QR) PayWithCard(ctx context.Context, cardToken string, in qr.PayQRInput) (*card.Purchase, error) {
	return s.payWithCard.Handle(ctx, qrcmd.PayQRCommand{CardToken: cardToken, PayQRInput: in})
}
