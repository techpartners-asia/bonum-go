// Package httpapi is the Gateway's outbound HTTP adapter. It implements every interface in
// gateway/ports against Bonum's gateway endpoints and owns everything transport-specific:
// the bearer token lifecycle, common headers, the mpay-service envelope and error decoding.
package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/techpartners-asia/bonum-go/internal/rest"
	"resty.dev/v3"
)

const (
	ecommercePath   = "/bonum-gateway/ecommerce"
	mpayPath        = "/mpay-service/merchant"
	cardTokenHeader = "X-CARD-TOKEN"
	defaultTimeout  = 30 * time.Second
	defaultLanguage = "mn"
)

// Client talks to the Bonum gateway on behalf of one merchant Terminal. It is safe for
// concurrent use; access tokens are fetched lazily and refreshed automatically.
type Client struct {
	rest   *rest.Client
	tokens *tokenSource
	lang   string
}

// New creates an adapter for baseURL authenticated with the Terminal's AppSecret.
func New(baseURL, appSecret, terminalID string) *Client {
	r := rest.New(baseURL, defaultTimeout, decodeError)
	return &Client{rest: r, tokens: newTokenSource(r, appSecret, terminalID), lang: defaultLanguage}
}

func (c *Client) SetBaseURL(u string)               { c.rest.SetBaseURL(u) }
func (c *Client) SetTimeout(d time.Duration)        { c.rest.SetTimeout(d) }
func (c *Client) SetTransport(rt http.RoundTripper) { c.rest.SetTransport(rt) }
func (c *Client) SetLanguage(lang string)           { c.lang = lang }
func (c *Client) Close() error                      { return c.rest.Close() }

// envelope is the wrapper every mpay-service endpoint returns. It is a transport detail:
// callers get Data and never see it.
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
	token, err := c.tokens.Token(ctx)
	if err != nil {
		return nil, err
	}
	req := c.rest.R().
		SetContext(ctx).
		SetHeader("Accept-Language", c.lang).
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
