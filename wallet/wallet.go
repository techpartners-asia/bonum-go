// Package wallet wraps the Bonum PSP V2 API for Apple Pay and Google Pay.
//
// The V2 API is separate from the gateway API wrapped by the root bonum package:
// different hosts, an x-merchant-key credential instead of AppSecret/terminal,
// and an asynchronous result model. Your mobile app or web page collects the
// encrypted wallet token; this package is for the backend that forwards it.
//
// This package is a facade: it composes internal/wallet/adapters/httpapi into the use cases
// in internal/wallet/application and re-exports the domain types so callers import only
// wallet. The internal packages are not importable outside this module by design.
package wallet

import (
	"context"
	"net/http"
	"time"

	"github.com/techpartners-asia/bonum-go/internal/wallet/adapters/httpapi"
	"github.com/techpartners-asia/bonum-go/internal/wallet/application"
)

// Environment selects which Bonum PSP host the client talks to.
type Environment string

const (
	Sandbox    Environment = "https://testpsp.bonum.mn"
	Production Environment = "https://psp.bonum.mn"
)

// MerchantKeyHeader authenticates every request.
const MerchantKeyHeader = httpapi.MerchantKeyHeader

// Client talks to the Bonum PSP V2 API on behalf of one merchant. It is safe for concurrent use.
type Client struct {
	payments *application.Payments
	api      *httpapi.Client
}

type Option func(*Client)

// WithBaseURL overrides the host derived from the Environment, e.g. to go through a proxy.
func WithBaseURL(baseURL string) Option { return func(c *Client) { c.api.SetBaseURL(baseURL) } }

// WithTimeout sets the per-request HTTP timeout (default 35s, which covers the 28s await cap).
func WithTimeout(d time.Duration) Option { return func(c *Client) { c.api.SetTimeout(d) } }

// WithTransport replaces the HTTP transport, e.g. to record outbound calls.
func WithTransport(rt http.RoundTripper) Option { return func(c *Client) { c.api.SetTransport(rt) } }

// New creates a Client. merchantKey is the Merchant Key issued by Bonum at onboarding.
func New(env Environment, merchantKey string, opts ...Option) *Client {
	api := httpapi.New(string(env), merchantKey)
	c := &Client{api: api}
	for _, opt := range opts {
		opt(c)
	}
	c.payments = application.NewPayments(api)
	return c
}

// Close releases the underlying HTTP resources.
func (c *Client) Close() error { return c.api.Close() }

// ProcessApplePay submits an Apple Pay token. The response is always PENDING; call
// AwaitPayment (or AwaitURL) to learn the outcome in time to close the wallet sheet.
func (c *Client) ProcessApplePay(ctx context.Context, in ProcessApplePayInput) (*ProcessResponse, error) {
	return c.payments.ProcessApplePay(ctx, in)
}

// ProcessGooglePay submits a Google Pay token string. See ProcessApplePay for the result model.
func (c *Client) ProcessGooglePay(ctx context.Context, in ProcessGooglePayInput) (*ProcessResponse, error) {
	return c.payments.ProcessGooglePay(ctx, in)
}

// GetPayment returns the current state of a Wallet Payment by Bonum's paymentId.
func (c *Client) GetPayment(ctx context.Context, paymentID string) (*Payment, error) {
	return c.payments.GetPayment(ctx, paymentID)
}

// LookupByOrderID returns the Wallet Payment submitted with your Order ID.
func (c *Client) LookupByOrderID(ctx context.Context, orderID string) (*Payment, error) {
	return c.payments.LookupByOrderID(ctx, orderID)
}

// AwaitPayment blocks until the payment reaches AUTHORIZED or FAILED, or timeout elapses.
// timeout 0 uses Bonum's default (25s); anything above MaxAwaitTimeout is capped at 28s.
// On TimedOut the payment is still processing and the outcome arrives via the webhook.
func (c *Client) AwaitPayment(ctx context.Context, paymentID string, timeout time.Duration) (*AwaitResult, error) {
	return c.payments.AwaitPayment(ctx, paymentID, timeout)
}

// AwaitURL is AwaitPayment for the absolute awaitUrl returned by ProcessApplePay / ProcessGooglePay.
func (c *Client) AwaitURL(ctx context.Context, awaitURL string, timeout time.Duration) (*AwaitResult, error) {
	return c.payments.AwaitURL(ctx, awaitURL, timeout)
}
