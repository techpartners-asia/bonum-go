// Package sandbox holds the Sandbox aggregate's write use cases: helpers Bonum only permits
// outside production.
package sandbox

import (
	"context"

	"github.com/techpartners-asia/bonum-go/internal/gateway/domain"
	"github.com/techpartners-asia/bonum-go/internal/gateway/ports"
)

// MarkInvoicePaidCommand marks InvoiceID as paid so the webhook fires.
type MarkInvoicePaidCommand struct{ InvoiceID string }

type MarkInvoicePaidHandler struct{ api ports.SandboxAPI }

func NewMarkInvoicePaidHandler(api ports.SandboxAPI) *MarkInvoicePaidHandler {
	return &MarkInvoicePaidHandler{api: api}
}

func (h *MarkInvoicePaidHandler) Handle(ctx context.Context, cmd MarkInvoicePaidCommand) error {
	if cmd.InvoiceID == "" {
		return domain.Invalid("invoiceID", "required")
	}
	return h.api.MarkInvoicePaid(ctx, cmd.InvoiceID)
}
