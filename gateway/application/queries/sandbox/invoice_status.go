// Package sandbox holds the Sandbox aggregate's read use case.
package sandbox

import (
	"context"
	"encoding/json"

	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// InvoiceStatusQuery returns the raw invoice record. Bonum forbids polling in production:
// rely on your own invoice table plus the webhook instead.
type InvoiceStatusQuery struct{ InvoiceID string }

type InvoiceStatusHandler struct{ api ports.SandboxAPI }

func NewInvoiceStatusHandler(api ports.SandboxAPI) *InvoiceStatusHandler {
	return &InvoiceStatusHandler{api: api}
}

func (h *InvoiceStatusHandler) Handle(ctx context.Context, q InvoiceStatusQuery) (json.RawMessage, error) {
	return h.api.InvoiceStatus(ctx, q.InvoiceID)
}
