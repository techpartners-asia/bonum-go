package httpapi

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

var _ ports.SandboxAPI = (*Client)(nil)

func (c *Client) InvoiceStatus(ctx context.Context, invoiceID string) (json.RawMessage, error) {
	out, err := call[json.RawMessage](ctx, c, http.MethodGet, ecommercePath+"/invoices/{id}", pathParam("id", invoiceID))
	if err != nil {
		return nil, err
	}
	return *out, nil
}

func (c *Client) MarkInvoicePaid(ctx context.Context, invoiceID string) error {
	_, err := call[json.RawMessage](ctx, c, http.MethodGet, ecommercePath+"/invoices/paid", query("invoiceId", invoiceID))
	return err
}

func (c *Client) RunSubscriptionBilling(ctx context.Context, id int64) error {
	return callAction(ctx, c, http.MethodPut, mpayPath+"/subscriptions/{id}/execute", subscriptionID(id))
}
