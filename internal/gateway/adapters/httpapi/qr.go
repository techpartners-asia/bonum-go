package httpapi

import (
	"context"
	"net/http"

	"github.com/techpartners-asia/bonum-go/internal/gateway/domain/card"
	"github.com/techpartners-asia/bonum-go/internal/gateway/domain/qr"
	"github.com/techpartners-asia/bonum-go/internal/gateway/ports"
)

var _ ports.QRAPI = (*Client)(nil)

func (c *Client) CreateQR(ctx context.Context, in qr.CreateQRInput) (*qr.QRInvoice, error) {
	return callEnveloped[qr.QRInvoice](ctx, c, http.MethodPost, mpayPath+"/transaction/qr/create", body(in))
}

func (c *Client) LookupQR(ctx context.Context, qrCode string) (*qr.QRInvoice, error) {
	return callEnveloped[qr.QRInvoice](ctx, c, http.MethodPost, mpayPath+"/transaction/qr", body(map[string]string{"qrCode": qrCode}))
}

func (c *Client) PayQRWithCard(ctx context.Context, cardTok string, in qr.PayQRInput) (*card.Purchase, error) {
	return callEnveloped[card.Purchase](ctx, c, http.MethodPut, mpayPath+"/transaction/qr/pay", cardToken(cardTok), body(in))
}
