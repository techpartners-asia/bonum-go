package bonum

import (
	"encoding/json"
	"net/http"

	"github.com/techpartners-asia/bonum-go/types"
)

// GetPaymentProviders lists the payment options currently enabled for this merchant.
func (c *Client) GetPaymentProviders() ([]types.PaymentProviderStatus, error) {
	req, err := c.authedRequest()
	if err != nil {
		return nil, err
	}
	var out []types.PaymentProviderStatus
	if err := c.do(req, http.MethodGet, ecommercePath+"/invoices/payment-providers", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateInvoice creates a checkout invoice. Redirect the customer to FollowUpLink; the final
// result arrives on the merchant webhook as a PAYMENT message.
func (c *Client) CreateInvoice(input types.CreateInvoiceInput) (*types.CreateInvoiceResponse, error) {
	req, err := c.authedRequest()
	if err != nil {
		return nil, err
	}
	var out types.CreateInvoiceResponse
	if err := c.do(req.SetBody(input), http.MethodPost, ecommercePath+"/invoices", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetInvoiceStatusSandbox returns the raw invoice record. Bonum explicitly forbids using this
// in production: rely on your own invoice table plus webhook delivery instead.
func (c *Client) GetInvoiceStatusSandbox(invoiceID string) (json.RawMessage, error) {
	req, err := c.authedRequest()
	if err != nil {
		return nil, err
	}
	var out json.RawMessage
	if err := c.do(req, http.MethodGet, ecommercePath+"/invoices/"+invoiceID, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// SetInvoicePaidSandbox marks an invoice as paid so the webhook fires. Sandbox only.
func (c *Client) SetInvoicePaidSandbox(invoiceID string) (json.RawMessage, error) {
	req, err := c.authedRequest()
	if err != nil {
		return nil, err
	}
	var out json.RawMessage
	if err := c.do(req.SetQueryParam("invoiceId", invoiceID), http.MethodGet, ecommercePath+"/invoices/paid", &out); err != nil {
		return nil, err
	}
	return out, nil
}
