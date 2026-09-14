// Package rest is the HTTP execution module shared by the gateway and wallet clients.
//
// Its interface is deliberately small: build a request, execute it against a path or
// absolute URL, decode a 2xx body into a value or a non-2xx body into the caller's error type.
// Host selection, timeouts and transport swapping live here so the API packages only
// describe endpoints.
package rest

import (
	"net/http"
	"strings"
	"time"

	"resty.dev/v3"
)

// ErrorFunc turns a non-2xx response into the package-specific error value.
type ErrorFunc func(status int, body string) error

type Client struct {
	baseURL string
	http    *resty.Client
	onError ErrorFunc
}

func New(baseURL string, timeout time.Duration, onError ErrorFunc) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    resty.New().SetTimeout(timeout).SetMethodDeleteAllowPayload(true),
		onError: onError,
	}
}

func (c *Client) BaseURL() string                   { return c.baseURL }
func (c *Client) SetBaseURL(u string)               { c.baseURL = strings.TrimRight(u, "/") }
func (c *Client) SetTimeout(d time.Duration)        { c.http.SetTimeout(d) }
func (c *Client) SetTransport(rt http.RoundTripper) { c.http.SetTransport(rt) }
func (c *Client) Close() error                      { return c.http.Close() }

// R starts a request that accepts JSON.
func (c *Client) R() *resty.Request {
	return c.http.R().SetHeader("Accept", "application/json")
}

// Do executes req. path is joined to the base URL unless it is already absolute.
// A 2xx body is decoded into result (when non-nil); anything else becomes onError(status, body).
func (c *Client) Do(req *resty.Request, method, path string, result any) error {
	if result != nil {
		req.SetResult(result)
	}
	if !isAbsolute(path) {
		path = c.baseURL + path
	}
	res, err := req.Execute(method, path)
	if err != nil {
		return err
	}
	if res.StatusCode() >= http.StatusBadRequest {
		return c.onError(res.StatusCode(), res.String())
	}
	return nil
}

func isAbsolute(u string) bool {
	return strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")
}
