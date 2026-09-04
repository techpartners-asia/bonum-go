package bonum

import (
	"net/http"

	"github.com/techpartners-asia/bonum-go/types"
)

// CreateQrCode creates a QPay-compatible QR invoice. Render QrImage for scanning or offer
// the per-bank Links as deeplinks on mobile. The result arrives as a PAYMENT webhook.
func (c *Client) CreateQrCode(input types.CreateQrCodeInput) (*types.CreateQrCodeResponse, error) {
	req, err := c.authedRequest()
	if err != nil {
		return nil, err
	}
	var out types.CreateQrCodeResponse
	if err := c.do(req.SetBody(input), http.MethodPost, mpayPath+"/transaction/qr/create", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// InvoiceByQrCode looks up the invoice behind a scanned QPay QR string.
func (c *Client) InvoiceByQrCode(qrCode string) (*types.InvoiceByQrCodeResponse, error) {
	req, err := c.authedRequest()
	if err != nil {
		return nil, err
	}
	var out types.InvoiceByQrCodeResponse
	if err := c.do(req.SetBody(map[string]string{"qrCode": qrCode}), http.MethodPost, mpayPath+"/transaction/qr", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// PayByCardToken settles a QR invoice with a stored card token.
func (c *Client) PayByCardToken(cardToken string, input types.PayByCardTokenInput) (*types.PayByCardTokenResponse, error) {
	req, err := c.authedRequest()
	if err != nil {
		return nil, err
	}
	var out types.PayByCardTokenResponse
	if err := c.do(req.SetHeader(cardTokenHeader, cardToken).SetBody(input), http.MethodPut, mpayPath+"/transaction/qr/pay", &out); err != nil {
		return nil, err
	}
	return &out, nil
}
