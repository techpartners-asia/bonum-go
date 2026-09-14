package application

import (
	"context"
	"encoding/json"

	sandboxcmd "github.com/techpartners-asia/bonum-go/gateway/application/commands/sandbox"
	sandboxqry "github.com/techpartners-asia/bonum-go/gateway/application/queries/sandbox"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// Sandbox groups helpers Bonum only permits outside production. Keeping them off the
// aggregate services stops them leaking into production code paths.
type Sandbox struct {
	invoiceStatus          *sandboxqry.InvoiceStatusHandler
	markInvoicePaid        *sandboxcmd.MarkInvoicePaidHandler
	runSubscriptionBilling *sandboxcmd.RunSubscriptionBillingHandler
}

func NewSandbox(api ports.SandboxAPI) *Sandbox {
	return &Sandbox{
		invoiceStatus:          sandboxqry.NewInvoiceStatusHandler(api),
		markInvoicePaid:        sandboxcmd.NewMarkInvoicePaidHandler(api),
		runSubscriptionBilling: sandboxcmd.NewRunSubscriptionBillingHandler(api),
	}
}

// InvoiceStatus returns the raw invoice record. Bonum forbids polling in production:
// rely on your own invoice table plus the webhook instead.
func (s *Sandbox) InvoiceStatus(ctx context.Context, invoiceID string) (json.RawMessage, error) {
	return s.invoiceStatus.Handle(ctx, sandboxqry.InvoiceStatusQuery{InvoiceID: invoiceID})
}

// MarkInvoicePaid marks an Invoice as paid so the webhook fires.
func (s *Sandbox) MarkInvoicePaid(ctx context.Context, invoiceID string) error {
	return s.markInvoicePaid.Handle(ctx, sandboxcmd.MarkInvoicePaidCommand{InvoiceID: invoiceID})
}

// RunSubscriptionBilling triggers a billing run for a Subscription on demand.
func (s *Sandbox) RunSubscriptionBilling(ctx context.Context, id int64) error {
	return s.runSubscriptionBilling.Handle(ctx, sandboxcmd.RunSubscriptionBillingCommand{ID: id})
}
