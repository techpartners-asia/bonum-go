// Package checkout holds the Checkout aggregate's read use case: listing enabled providers.
package checkout

import (
	"context"

	"github.com/techpartners-asia/bonum-go/gateway/domain/checkout"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// ProvidersHandler lists the payment options currently enabled for the Terminal.
type ProvidersHandler struct{ api ports.CheckoutAPI }

func NewProvidersHandler(api ports.CheckoutAPI) *ProvidersHandler {
	return &ProvidersHandler{api: api}
}

// Handle takes no input: Providers always reads the calling Terminal's configuration.
func (h *ProvidersHandler) Handle(ctx context.Context) ([]checkout.PaymentProviderStatus, error) {
	return h.api.Providers(ctx)
}
