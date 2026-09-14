// Package qr holds the QR Invoice aggregate's write use cases.
package qr

import (
	"context"

	"github.com/techpartners-asia/bonum-go/gateway/domain/qr"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// CreateQRCommand opens a QR Invoice.
type CreateQRCommand = qr.CreateQRInput

// CreateQRHandler opens a QR Invoice. Render QrImage or offer Links; the outcome arrives as
// a PaymentEvent.
type CreateQRHandler struct{ api ports.QRAPI }

func NewCreateQRHandler(api ports.QRAPI) *CreateQRHandler { return &CreateQRHandler{api: api} }

func (h *CreateQRHandler) Handle(ctx context.Context, cmd CreateQRCommand) (*qr.QRInvoice, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}
	return h.api.CreateQR(ctx, cmd)
}
