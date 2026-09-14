// Package bonum is the Gateway bounded context of the Bonum payment SDK: invoices, card
// tokens, purchases, subscriptions, QR invoices and the gateway webhook.
//
// It is backend only. The AppSecret, checksum key and bearer tokens must never reach a
// browser or mobile app. Apple Pay / Google Pay live in the separate wallet package.
//
// This package is a facade: it composes internal/gateway/adapters/httpapi into the use cases
// in internal/gateway/application and re-exports the domain types so callers import only
// bonum. The internal packages are not importable outside this module by design.
package bonum

import (
	"context"
	"net/http"
	"time"

	"github.com/techpartners-asia/bonum-go/internal/gateway/adapters/httpapi"
	"github.com/techpartners-asia/bonum-go/internal/gateway/application"
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

	access *application.Access
	api    *httpapi.Client
}

type Option func(*Client)

// WithLanguage sets the Accept-Language header (default MN).
func WithLanguage(lang Lang) Option { return func(c *Client) { c.api.SetLanguage(string(lang)) } }

// WithBaseURL overrides the host derived from the Environment, e.g. to go through a proxy.
func WithBaseURL(baseURL string) Option { return func(c *Client) { c.api.SetBaseURL(baseURL) } }

// WithTimeout sets the per-request HTTP timeout (default 30s).
func WithTimeout(d time.Duration) Option { return func(c *Client) { c.api.SetTimeout(d) } }

// WithTransport replaces the HTTP transport, e.g. to record outbound calls.
func WithTransport(rt http.RoundTripper) Option { return func(c *Client) { c.api.SetTransport(rt) } }

// New creates a Client. appSecret and terminalID come from the Bonum merchant portal.
func New(env Environment, appSecret, terminalID string, opts ...Option) *Client {
	api := httpapi.New(string(env), appSecret, terminalID)
	c := &Client{api: api}
	for _, opt := range opts {
		opt(c)
	}
	c.access = application.NewAccess(api)
	c.Invoices = application.NewInvoices(api)
	c.Cards = application.NewCards(api)
	c.Subscriptions = application.NewSubscriptions(api)
	c.QR = application.NewQR(api)
	c.Sandbox = application.NewSandbox(api)
	return c
}

// Close releases the underlying HTTP resources.
func (c *Client) Close() error { return c.api.Close() }

// Authenticate forces a fresh TokenPair via auth/create. Normally unnecessary: every call
// obtains a token on demand. The endpoint is rate limited; do not call it in a loop.
func (c *Client) Authenticate(ctx context.Context) (*TokenPair, error) { return c.access.Authenticate(ctx) }

// Refresh exchanges the cached refresh token for a new access token via auth/refresh.
func (c *Client) Refresh(ctx context.Context) (*TokenPair, error) { return c.access.Refresh(ctx) }
