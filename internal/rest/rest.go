// Package rest is the HTTP execution module shared by the gateway and wallet clients.
//
// Its interface is deliberately small: build a request, execute it against a path or
// absolute URL, decode a 2xx body into a value or a non-2xx body into the caller's error type.
// Host selection, timeouts and transport swapping live here so the API packages only
// describe endpoints.
package rest

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
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
	strict  bool
}

// Option adjusts a Client at construction.
type Option func(*Client)

// StrictDecoding makes a 2xx body that is empty or not JSON for the result a *DecodeError
// instead of a zero-valued result. The wallet API documents a JSON body on every 2xx.
func StrictDecoding() Option { return func(c *Client) { c.strict = true } }

func New(baseURL string, timeout time.Duration, onError ErrorFunc, opts ...Option) *Client {
	c := &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    resty.New().SetTimeout(timeout).SetMethodDeleteAllowPayload(true),
		onError: onError,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// DecodeError is a 2xx response whose body could not be decoded into the result.
type DecodeError struct {
	StatusCode  int
	ContentType string
	Body        string
	Err         error // nil when the body was empty
}

func (e *DecodeError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("%d response with an empty body", e.StatusCode)
	}
	return fmt.Sprintf("%d response not decodable (%s): %v", e.StatusCode, e.ContentType, e.Err)
}

func (e *DecodeError) Unwrap() error { return e.Err }

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
//
// The body is decoded as JSON whatever its Content-Type. resty's SetResult decodes only a
// body labelled JSON and silently leaves result zero otherwise, which turned a live
// /process/google answer into an empty paymentId with no error.
func (c *Client) Do(req *resty.Request, method, path string, result any) error {
	if !isAbsolute(path) {
		path = c.baseURL + path
	}
	res, err := req.Execute(method, path)
	if err != nil {
		return err
	}
	data, decodeErr := inflate(res.Bytes())
	if res.StatusCode() >= http.StatusBadRequest {
		return c.onError(res.StatusCode(), strings.TrimSpace(string(data)))
	}
	if result == nil {
		return nil
	}
	data = bytes.TrimSpace(data)
	if decodeErr == nil && len(data) > 0 {
		decodeErr = json.Unmarshal(data, result)
		if decodeErr == nil {
			return nil
		}
	}
	if !c.strict {
		return nil
	}
	return &DecodeError{
		StatusCode:  res.StatusCode(),
		ContentType: res.Header().Get("Content-Type"),
		Body:        string(data),
		Err:         decodeErr,
	}
}

// maxInflated bounds a decompressed body; every documented response is a few hundred bytes.
const maxInflated = 1 << 20

// inflate decompresses a gzip body that arrived without Content-Encoding. resty sends
// its own Accept-Encoding, which turns off net/http's transparent decompression, and
// inflates only when Content-Encoding says gzip; live /process/google answered gzip with
// no such header. A body without the gzip magic number is returned unchanged.
func inflate(data []byte) ([]byte, error) {
	if len(data) < 2 || data[0] != 0x1f || data[1] != 0x8b {
		return data, nil
	}
	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return data, fmt.Errorf("gzip body: %w", err)
	}
	defer zr.Close()
	out, err := io.ReadAll(io.LimitReader(zr, maxInflated+1))
	switch {
	case err != nil:
		return data, fmt.Errorf("gzip body: %w", err)
	case len(out) > maxInflated:
		return data, fmt.Errorf("gzip body larger than %d bytes", maxInflated)
	}
	return out, nil
}

func isAbsolute(u string) bool {
	return strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")
}
