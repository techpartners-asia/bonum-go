package bonum

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/techpartners-asia/bonum-go/types"
	"resty.dev/v3"
)

const (
	ecommercePath = "/bonum-gateway/ecommerce"
	mpayPath      = "/mpay-service/merchant"

	// Refresh a little early so an in-flight request never races the server-side expiry.
	tokenExpirySkew = 30 * time.Second
)

// Client talks to the Bonum gateway on behalf of one merchant terminal.
// It is safe for concurrent use; access tokens are fetched lazily and refreshed automatically.
type Client struct {
	baseURL    string
	appSecret  string
	terminalID string
	lang       types.Lang
	http       *resty.Client

	mu               sync.Mutex
	accessToken      string
	refreshToken     string
	accessExpiresAt  time.Time
	refreshExpiresAt time.Time
}

type Option func(*Client)

// WithLanguage sets the Accept-Language header (default MN).
func WithLanguage(lang types.Lang) Option {
	return func(c *Client) { c.lang = lang }
}

// WithBaseURL overrides the host derived from the Environment, e.g. to go through a proxy.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) { c.baseURL = strings.TrimRight(baseURL, "/") }
}

// WithTimeout sets the per-request HTTP timeout (default 30s).
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.http.SetTimeout(d) }
}

// WithTransport replaces the HTTP transport, e.g. to record outbound calls.
func WithTransport(rt http.RoundTripper) Option {
	return func(c *Client) { c.http.SetTransport(rt) }
}

// New creates a Client. appSecret and terminalID come from the Bonum merchant portal.
func New(env types.Environment, appSecret, terminalID string, opts ...Option) *Client {
	c := &Client{
		baseURL:    string(env),
		appSecret:  appSecret,
		terminalID: terminalID,
		lang:       types.MN,
		http:       resty.New().SetTimeout(30 * time.Second).SetMethodDeleteAllowPayload(true),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Close releases the underlying HTTP resources.
func (c *Client) Close() error {
	return c.http.Close()
}

// Error is returned when Bonum answers with a non-2xx status.
type Error struct {
	StatusCode int
	TraceID    string
	Message    string
	Body       string
}

func (e *Error) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("bonum: %d %s (trace %s)", e.StatusCode, e.Message, e.TraceID)
	}
	return fmt.Sprintf("bonum: %d %s", e.StatusCode, e.Body)
}

// authedRequest builds a request carrying a valid bearer token plus the common headers.
func (c *Client) authedRequest() (*resty.Request, error) {
	token, err := c.token()
	if err != nil {
		return nil, err
	}
	return c.http.R().
		SetHeader("Accept", "application/json").
		SetHeader("Accept-Language", string(c.lang)).
		SetAuthToken(token), nil
}

// do executes req against baseURL+path, decodes a 2xx body into result and a non-2xx body into *Error.
func (c *Client) do(req *resty.Request, method, path string, result any) error {
	if result != nil {
		req.SetResult(result)
	}
	res, err := req.Execute(method, c.baseURL+path)
	if err != nil {
		return err
	}
	if res.StatusCode() >= http.StatusBadRequest {
		return newError(res)
	}
	return nil
}

func newError(res *resty.Response) *Error {
	body := res.String()
	e := &Error{StatusCode: res.StatusCode(), Body: body}
	var er types.ErrorResponse
	if json.Unmarshal([]byte(body), &er) == nil {
		e.TraceID = er.TraceID
		e.Message = er.Message
	}
	return e
}
