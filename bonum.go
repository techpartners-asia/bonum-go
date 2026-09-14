// Package bonum is the Gateway bounded context of the Bonum payment SDK: invoices, card
// tokens, purchases, subscriptions, QR invoices and the gateway webhook.
//
// It is backend only. The AppSecret, checksum key and bearer tokens must never reach a
// browser or mobile app. Apple Pay / Google Pay live in the separate wallet package.
package bonum

import (
	"context"
	"net/http"
	"time"

	"github.com/techpartners-asia/bonum-go/internal/rest"
	"resty.dev/v3"
)

// Environment selects which Bonum gateway host the client talks to.
type Environment string

const (
	Sandbox    Environment = "https://testapi.bonum.mn"
	Production Environment = "https://apis.bonum.mn"
)

// Lang is sent as Accept-Language so Bonum localises response messages.
type Lang string

const (
	MN Lang = "mn"
	EN Lang = "en"
)

const (
	ecommercePath   = "/bonum-gateway/ecommerce"
	mpayPath        = "/mpay-service/merchant"
	cardTokenHeader = "X-CARD-TOKEN"
)

// Client talks to the Bonum gateway on behalf of one merchant Terminal. It is safe for
// concurrent use; access tokens are fetched lazily and refreshed automatically.
//
// Operations are grouped by aggregate: Invoices, Cards, Subscriptions, QR. Sandbox holds
// helpers Bonum only allows outside production.
type Client struct {
	Invoices      *InvoiceService
	Cards         *CardService
	Subscriptions *SubscriptionService
	QR            *QRService
	Sandbox       *SandboxService

	rest *rest.Client
	auth *tokenSource
	lang Lang
}

type Option func(*Client)

// WithLanguage sets the Accept-Language header (default MN).
func WithLanguage(lang Lang) Option { return func(c *Client) { c.lang = lang } }

// WithBaseURL overrides the host derived from the Environment, e.g. to go through a proxy.
func WithBaseURL(baseURL string) Option { return func(c *Client) { c.rest.SetBaseURL(baseURL) } }

// WithTimeout sets the per-request HTTP timeout (default 30s).
func WithTimeout(d time.Duration) Option { return func(c *Client) { c.rest.SetTimeout(d) } }

// WithTransport replaces the HTTP transport, e.g. to record outbound calls.
func WithTransport(rt http.RoundTripper) Option { return func(c *Client) { c.rest.SetTransport(rt) } }

// New creates a Client. appSecret and terminalID come from the Bonum merchant portal.
func New(env Environment, appSecret, terminalID string, opts ...Option) *Client {
	r := rest.New(string(env), 30*time.Second, newAPIError)
	c := &Client{rest: r, auth: newTokenSource(r, appSecret, terminalID), lang: MN}
	for _, opt := range opts {
		opt(c)
	}
	c.Invoices = &InvoiceService{c}
	c.Cards = &CardService{c}
	c.Subscriptions = &SubscriptionService{c}
	c.QR = &QRService{c}
	c.Sandbox = &SandboxService{c}
	return c
}

// Close releases the underlying HTTP resources.
func (c *Client) Close() error { return c.rest.Close() }

// Authenticate forces a fresh TokenPair via auth/create. Normally unnecessary: every call
// obtains a token on demand. The endpoint is rate limited; do not call it in a loop.
func (c *Client) Authenticate(ctx context.Context) (*TokenPair, error) { return c.auth.Create(ctx) }

// Refresh exchanges the cached refresh token for a new access token via auth/refresh.
func (c *Client) Refresh(ctx context.Context) (*TokenPair, error) { return c.auth.Refresh(ctx) }

// --- transport helpers shared by the services -------------------------------------------

// envelope is the wrapper every mpay-service endpoint returns. It is a transport detail:
// services unwrap Data and never expose it.
type envelope[T any] struct {
	TraceID string `json:"traceId"`
	Message string `json:"message"`
	Data    T      `json:"data"`
	Status  int    `json:"status"`
}

type reqOpt func(*resty.Request)

func body(v any) reqOpt { return func(r *resty.Request) { r.SetBody(v) } }
func cardToken(tok string) reqOpt {
	return func(r *resty.Request) { r.SetHeader(cardTokenHeader, tok) }
}
func pathParam(k, v string) reqOpt { return func(r *resty.Request) { r.SetPathParam(k, v) } }
func query(k, v string) reqOpt     { return func(r *resty.Request) { r.SetQueryParam(k, v) } }

// call is the single path every gateway endpoint goes through: obtain a bearer token, add
// the common headers, apply the endpoint's options, execute, decode into T.
func call[T any](ctx context.Context, c *Client, method, path string, opts ...reqOpt) (*T, error) {
	token, err := c.auth.Token(ctx)
	if err != nil {
		return nil, err
	}
	req := c.rest.R().
		SetContext(ctx).
		SetHeader("Accept-Language", string(c.lang)).
		SetAuthToken(token)
	for _, opt := range opts {
		opt(req)
	}
	var out T
	if err := c.rest.Do(req, method, path, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// callEnveloped is call for mpay-service endpoints; it returns the unwrapped Data.
func callEnveloped[T any](ctx context.Context, c *Client, method, path string, opts ...reqOpt) (*T, error) {
	env, err := call[envelope[T]](ctx, c, method, path, opts...)
	if err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// callAction is callEnveloped for endpoints whose Data carries nothing useful.
func callAction(ctx context.Context, c *Client, method, path string, opts ...reqOpt) error {
	_, err := call[envelope[any]](ctx, c, method, path, opts...)
	return err
}
