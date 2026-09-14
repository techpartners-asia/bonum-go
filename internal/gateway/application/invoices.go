package application

import (
	"context"

	checkoutcmd "github.com/techpartners-asia/bonum-go/internal/gateway/application/commands/checkout"
	checkoutqry "github.com/techpartners-asia/bonum-go/internal/gateway/application/queries/checkout"
	"github.com/techpartners-asia/bonum-go/internal/gateway/domain/checkout"
	"github.com/techpartners-asia/bonum-go/internal/gateway/ports"
)

// Invoices is the Invoice aggregate's use cases: hosted checkout.
type Invoices struct {
	providers *checkoutqry.ProvidersHandler
	create    *checkoutcmd.CreateInvoiceHandler
}

func NewInvoices(api ports.CheckoutAPI) *Invoices {
	return &Invoices{
		providers: checkoutqry.NewProvidersHandler(api),
		create:    checkoutcmd.NewCreateInvoiceHandler(api),
	}
}

// Providers lists the payment options currently enabled for this Terminal.
func (s *Invoices) Providers(ctx context.Context) ([]checkout.PaymentProviderStatus, error) {
	return s.providers.Handle(ctx)
}

// Create opens an Invoice. Redirect the customer to FollowUpLink.
func (s *Invoices) Create(ctx context.Context, in checkout.CreateInvoiceInput) (*checkout.Invoice, error) {
	return s.create.Handle(ctx, in)
}
