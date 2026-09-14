// Package httpapi is the Wallet's outbound HTTP adapter: it implements wallet/ports against
// Bonum's V2 endpoints and owns the merchant-key header, timeouts and error decoding.
package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/techpartners-asia/bonum-go/internal/rest"
	"github.com/techpartners-asia/bonum-go/internal/wallet/domain/payment"
	"github.com/techpartners-asia/bonum-go/internal/wallet/ports"
	"resty.dev/v3"
)

var _ ports.PaymentAPI = (*Client)(nil)

const (
	processApplePath  = "/api/v2/payment/process"
	processGooglePath = "/api/v2/payment/process/google"
	paymentsPath      = "/api/v2/payments"

	// MerchantKeyHeader authenticates every request.
	MerchantKeyHeader = "x-merchant-key"

	// The await endpoint blocks up to 28s; leave room for network latency on top.
	defaultTimeout = 35 * time.Second
)

// Client talks to the Bonum PSP V2 API on behalf of one merchant. It is safe for concurrent use.
type Client struct {
	rest        *rest.Client
	merchantKey string
}

// New creates an adapter for baseURL authenticated with the Merchant Key.
func New(baseURL, merchantKey string) *Client {
	return &Client{rest: rest.New(baseURL, defaultTimeout, decodeError), merchantKey: merchantKey}
}

func (c *Client) SetBaseURL(u string)               { c.rest.SetBaseURL(u) }
func (c *Client) SetTimeout(d time.Duration)        { c.rest.SetTimeout(d) }
func (c *Client) SetTransport(rt http.RoundTripper) { c.rest.SetTransport(rt) }
func (c *Client) Close() error                      { return c.rest.Close() }

func (c *Client) ProcessApplePay(ctx context.Context, in payment.ProcessApplePayInput) (*payment.ProcessResponse, error) {
	return call[payment.ProcessResponse](ctx, c, http.MethodPost, processApplePath, body(in))
}

func (c *Client) ProcessGooglePay(ctx context.Context, in payment.ProcessGooglePayInput) (*payment.ProcessResponse, error) {
	return call[payment.ProcessResponse](ctx, c, http.MethodPost, processGooglePath, body(in))
}

func (c *Client) GetPayment(ctx context.Context, paymentID string) (*payment.Payment, error) {
	return call[payment.Payment](ctx, c, http.MethodGet, paymentsPath+"/{id}", pathParam("id", paymentID))
}

func (c *Client) LookupByOrderID(ctx context.Context, orderID string) (*payment.Payment, error) {
	return call[payment.Payment](ctx, c, http.MethodGet, paymentsPath+"/lookup/by-order-id", query("orderId", orderID))
}

func (c *Client) AwaitPayment(ctx context.Context, paymentID string, timeout time.Duration) (*payment.AwaitResult, error) {
	return call[payment.AwaitResult](ctx, c, http.MethodGet, paymentsPath+"/{id}/await", pathParam("id", paymentID), awaitTimeout(timeout))
}

func (c *Client) AwaitURL(ctx context.Context, awaitURL string, timeout time.Duration) (*payment.AwaitResult, error) {
	return call[payment.AwaitResult](ctx, c, http.MethodGet, awaitURL, awaitTimeout(timeout))
}

type reqOpt func(*resty.Request)

func body(v any) reqOpt            { return func(r *resty.Request) { r.SetBody(v) } }
func query(k, v string) reqOpt     { return func(r *resty.Request) { r.SetQueryParam(k, v) } }
func pathParam(k, v string) reqOpt { return func(r *resty.Request) { r.SetPathParam(k, v) } }

// awaitTimeout encodes an already-clamped timeout; 0 omits the parameter so Bonum applies its default.
func awaitTimeout(d time.Duration) reqOpt {
	if d <= 0 {
		return func(*resty.Request) {}
	}
	return query("timeoutMs", strconv.FormatInt(d.Milliseconds(), 10))
}

// call is the single path every V2 endpoint goes through: merchant key, JSON headers,
// endpoint options, execute against a path or absolute URL, decode into T.
func call[T any](ctx context.Context, c *Client, method, path string, opts ...reqOpt) (*T, error) {
	req := c.rest.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetHeader(MerchantKeyHeader, c.merchantKey)
	for _, opt := range opts {
		opt(req)
	}
	var out T
	if err := c.rest.Do(req, method, path, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
