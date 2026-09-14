package httpapi

import (
	"context"
	"net/http"

	"github.com/techpartners-asia/bonum-go/internal/gateway/domain/checkout"
	"github.com/techpartners-asia/bonum-go/internal/gateway/ports"
)

var _ ports.CheckoutAPI = (*Client)(nil)

func (c *Client) Providers(ctx context.Context) ([]checkout.PaymentProviderStatus, error) {
	out, err := call[[]checkout.PaymentProviderStatus](ctx, c, http.MethodGet, ecommercePath+"/invoices/payment-providers")
	if err != nil {
		return nil, err
	}
	return *out, nil
}

func (c *Client) CreateInvoice(ctx context.Context, in checkout.CreateInvoiceInput) (*checkout.Invoice, error) {
	return call[checkout.Invoice](ctx, c, http.MethodPost, ecommercePath+"/invoices", body(in))
}
