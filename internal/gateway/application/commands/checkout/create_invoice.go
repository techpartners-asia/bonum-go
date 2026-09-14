// Package checkout holds the Checkout aggregate's write use case: opening an Invoice.
package checkout

import (
	"context"

	"github.com/techpartners-asia/bonum-go/internal/gateway/domain/checkout"
	"github.com/techpartners-asia/bonum-go/internal/gateway/ports"
)

// CreateInvoiceCommand is the Invoice aggregate's create input.
type CreateInvoiceCommand = checkout.CreateInvoiceInput

// CreateInvoiceHandler opens an Invoice. Redirect the customer to FollowUpLink.
type CreateInvoiceHandler struct{ api ports.CheckoutAPI }

func NewCreateInvoiceHandler(api ports.CheckoutAPI) *CreateInvoiceHandler {
	return &CreateInvoiceHandler{api: api}
}

func (h *CreateInvoiceHandler) Handle(ctx context.Context, cmd CreateInvoiceCommand) (*checkout.Invoice, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}
	return h.api.CreateInvoice(ctx, cmd)
}
