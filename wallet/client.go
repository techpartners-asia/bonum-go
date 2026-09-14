package wallet

import (
	"context"
	"net/http"
	"time"

	"github.com/techpartners-asia/bonum-go/internal/rest"
	"resty.dev/v3"
)

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

type Option func(*Client)

// WithBaseURL overrides the host derived from the Environment, e.g. to go through a proxy.
func WithBaseURL(baseURL string) Option { return func(c *Client) { c.rest.SetBaseURL(baseURL) } }

// WithTimeout sets the per-request HTTP timeout (default 35s, which covers the 28s await cap).
func WithTimeout(d time.Duration) Option { return func(c *Client) { c.rest.SetTimeout(d) } }

// WithTransport replaces the HTTP transport, e.g. to record outbound calls.
func WithTransport(rt http.RoundTripper) Option { return func(c *Client) { c.rest.SetTransport(rt) } }

// New creates a Client. merchantKey is the Merchant Key issued by Bonum at onboarding.
func New(env Environment, merchantKey string, opts ...Option) *Client {
	c := &Client{rest: rest.New(string(env), defaultTimeout, newAPIError), merchantKey: merchantKey}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Close releases the underlying HTTP resources.
func (c *Client) Close() error { return c.rest.Close() }

// ProcessApplePay submits an Apple Pay token. The response is always PENDING; call
// AwaitPayment (or AwaitURL) to learn the outcome in time to close the wallet sheet.
func (c *Client) ProcessApplePay(ctx context.Context, in ProcessApplePayInput) (*ProcessResponse, error) {
	if err := in.validate(); err != nil {
		return nil, err
	}
	return call[ProcessResponse](ctx, c, http.MethodPost, processApplePath, body(in))
}

// ProcessGooglePay submits a Google Pay token string. See ProcessApplePay for the result model.
func (c *Client) ProcessGooglePay(ctx context.Context, in ProcessGooglePayInput) (*ProcessResponse, error) {
	if err := in.validate(); err != nil {
		return nil, err
	}
	return call[ProcessResponse](ctx, c, http.MethodPost, processGooglePath, body(in))
}

type reqOpt func(*resty.Request)

func body(v any) reqOpt            { return func(r *resty.Request) { r.SetBody(v) } }
func query(k, v string) reqOpt     { return func(r *resty.Request) { r.SetQueryParam(k, v) } }
func pathParam(k, v string) reqOpt { return func(r *resty.Request) { r.SetPathParam(k, v) } }

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
