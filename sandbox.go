package bonum

import (
	"context"
	"encoding/json"
	"net/http"
)

// SandboxService groups helpers Bonum only permits outside production. Keeping them off the
// aggregate services stops them leaking into production code paths.
type SandboxService struct{ c *Client }

// InvoiceStatus returns the raw invoice record. Bonum forbids polling in production:
// rely on your own invoice table plus the webhook instead.
func (s *SandboxService) InvoiceStatus(ctx context.Context, invoiceID string) (json.RawMessage, error) {
	out, err := call[json.RawMessage](ctx, s.c, http.MethodGet, ecommercePath+"/invoices/{id}", pathParam("id", invoiceID))
	if err != nil {
		return nil, err
	}
	return *out, nil
}

// MarkInvoicePaid marks an Invoice as paid so the webhook fires.
func (s *SandboxService) MarkInvoicePaid(ctx context.Context, invoiceID string) error {
	_, err := call[json.RawMessage](ctx, s.c, http.MethodGet, ecommercePath+"/invoices/paid", query("invoiceId", invoiceID))
	return err
}

// RunSubscriptionBilling triggers a billing run for a Subscription on demand.
func (s *SandboxService) RunSubscriptionBilling(ctx context.Context, id int64) error {
	return callAction(ctx, s.c, http.MethodPut, mpayPath+"/subscriptions/{id}/execute", subscriptionID(id))
}
