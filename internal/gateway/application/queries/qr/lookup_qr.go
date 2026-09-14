// Package qr holds the QR Invoice aggregate's read use case.
package qr

import (
	"context"

	"github.com/techpartners-asia/bonum-go/internal/gateway/domain"
	"github.com/techpartners-asia/bonum-go/internal/gateway/domain/qr"
	"github.com/techpartners-asia/bonum-go/internal/gateway/ports"
)

// LookupQRQuery returns the QR Invoice behind a scanned QPay QR string.
type LookupQRQuery struct{ QrCode string }

type LookupQRHandler struct{ api ports.QRAPI }

func NewLookupQRHandler(api ports.QRAPI) *LookupQRHandler { return &LookupQRHandler{api: api} }

func (h *LookupQRHandler) Handle(ctx context.Context, q LookupQRQuery) (*qr.QRInvoice, error) {
	if q.QrCode == "" {
		return nil, domain.Invalid("qrCode", "required")
	}
	return h.api.LookupQR(ctx, q.QrCode)
}
