// Package application holds the Wallet facade: one struct exposing the same public methods
// as before, delegating to Command/Query Handlers in application/commands/payment and
// application/queries/payment.
package application

import (
	"context"
	"time"

	paymentcmd "github.com/techpartners-asia/bonum-go/wallet/application/commands/payment"
	paymentqry "github.com/techpartners-asia/bonum-go/wallet/application/queries/payment"
	"github.com/techpartners-asia/bonum-go/wallet/domain/payment"
	"github.com/techpartners-asia/bonum-go/wallet/ports"
)

// Payments is the Wallet Payment aggregate's use cases.
type Payments struct {
	processApplePay  *paymentcmd.ProcessApplePayHandler
	processGooglePay *paymentcmd.ProcessGooglePayHandler
	getPayment       *paymentqry.GetPaymentHandler
	lookupByOrderID  *paymentqry.LookupByOrderIDHandler
	awaitPayment     *paymentqry.AwaitPaymentHandler
	awaitURL         *paymentqry.AwaitURLHandler
}

func NewPayments(api ports.PaymentAPI) *Payments {
	return &Payments{
		processApplePay:  paymentcmd.NewProcessApplePayHandler(api),
		processGooglePay: paymentcmd.NewProcessGooglePayHandler(api),
		getPayment:       paymentqry.NewGetPaymentHandler(api),
		lookupByOrderID:  paymentqry.NewLookupByOrderIDHandler(api),
		awaitPayment:     paymentqry.NewAwaitPaymentHandler(api),
		awaitURL:         paymentqry.NewAwaitURLHandler(api),
	}
}

// ProcessApplePay submits an Apple Pay token. The response is always PENDING; call
// AwaitPayment (or AwaitURL) to learn the outcome in time to close the wallet sheet.
func (s *Payments) ProcessApplePay(ctx context.Context, in payment.ProcessApplePayInput) (*payment.ProcessResponse, error) {
	return s.processApplePay.Handle(ctx, in)
}

// ProcessGooglePay submits a Google Pay token string. See ProcessApplePay for the result model.
func (s *Payments) ProcessGooglePay(ctx context.Context, in payment.ProcessGooglePayInput) (*payment.ProcessResponse, error) {
	return s.processGooglePay.Handle(ctx, in)
}

// GetPayment returns the current state of a Wallet Payment by Bonum's paymentId.
func (s *Payments) GetPayment(ctx context.Context, paymentID string) (*payment.Payment, error) {
	return s.getPayment.Handle(ctx, paymentqry.GetPaymentQuery{PaymentID: paymentID})
}

// LookupByOrderID returns the Wallet Payment submitted with your Order ID.
func (s *Payments) LookupByOrderID(ctx context.Context, orderID string) (*payment.Payment, error) {
	return s.lookupByOrderID.Handle(ctx, paymentqry.LookupByOrderIDQuery{OrderID: orderID})
}

// AwaitPayment blocks until the payment reaches AUTHORIZED or FAILED, or timeout elapses.
// timeout 0 uses Bonum's default (25s); anything above MaxAwaitTimeout is capped at 28s.
// On TimedOut the payment is still processing and the outcome arrives via the webhook.
func (s *Payments) AwaitPayment(ctx context.Context, paymentID string, timeout time.Duration) (*payment.AwaitResult, error) {
	return s.awaitPayment.Handle(ctx, paymentqry.AwaitPaymentQuery{PaymentID: paymentID, Timeout: timeout})
}

// AwaitURL is AwaitPayment for the absolute awaitUrl returned by ProcessApplePay / ProcessGooglePay.
func (s *Payments) AwaitURL(ctx context.Context, awaitURL string, timeout time.Duration) (*payment.AwaitResult, error) {
	return s.awaitURL.Handle(ctx, paymentqry.AwaitURLQuery{AwaitURL: awaitURL, Timeout: timeout})
}
