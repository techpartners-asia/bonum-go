package qr

import (
	"context"

	"github.com/techpartners-asia/bonum-go/internal/gateway/domain/card"
	"github.com/techpartners-asia/bonum-go/internal/gateway/domain/qr"
	"github.com/techpartners-asia/bonum-go/internal/gateway/ports"
)

// PayQRCommand settles a QR Invoice with a stored Card Token.
type PayQRCommand struct {
	CardToken string
	qr.PayQRInput
}

type PayQRHandler struct{ api ports.QRAPI }

func NewPayQRHandler(api ports.QRAPI) *PayQRHandler { return &PayQRHandler{api: api} }

func (h *PayQRHandler) Handle(ctx context.Context, cmd PayQRCommand) (*card.Purchase, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}
	return h.api.PayQRWithCard(ctx, cmd.CardToken, cmd.PayQRInput)
}
