// Package ports declares the outbound capability the Wallet application layer depends on,
// implemented by adapters/httpapi.
package ports

import (
	"context"
	"time"

	"github.com/techpartners-asia/bonum-go/wallet/domain/payment"
)

// PaymentAPI submits wallet tokens and reads Wallet Payments. Inputs arrive validated and
// timeouts arrive clamped; 0 means Bonum's server-side default.
type PaymentAPI interface {
	ProcessApplePay(ctx context.Context, in payment.ProcessApplePayInput) (*payment.ProcessResponse, error)
	ProcessGooglePay(ctx context.Context, in payment.ProcessGooglePayInput) (*payment.ProcessResponse, error)
	GetPayment(ctx context.Context, paymentID string) (*payment.Payment, error)
	LookupByOrderID(ctx context.Context, orderID string) (*payment.Payment, error)
	AwaitPayment(ctx context.Context, paymentID string, timeout time.Duration) (*payment.AwaitResult, error)
	AwaitURL(ctx context.Context, awaitURL string, timeout time.Duration) (*payment.AwaitResult, error)
}
