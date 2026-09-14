package httpapi

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/techpartners-asia/bonum-go/internal/gateway/domain/access"
	"github.com/techpartners-asia/bonum-go/internal/gateway/ports"
	"github.com/techpartners-asia/bonum-go/internal/rest"
	"resty.dev/v3"
)

var _ ports.AccessAPI = (*Client)(nil)

// Refresh a little early so an in-flight request never races the server-side expiry.
const tokenExpirySkew = 30 * time.Second

// tokenSource owns the AppSecret -> bearer token lifecycle. Its interface is Token():
// callers get a valid access token and never see create/refresh/expiry.
type tokenSource struct {
	rest       *rest.Client
	appSecret  string
	terminalID string
	now        func() time.Time

	mu               sync.Mutex
	accessToken      string
	refreshToken     string
	accessExpiresAt  time.Time
	refreshExpiresAt time.Time
}

func newTokenSource(r *rest.Client, appSecret, terminalID string) *tokenSource {
	return &tokenSource{rest: r, appSecret: appSecret, terminalID: terminalID, now: time.Now}
}

// Token returns a usable access token, refreshing or re-authenticating as needed.
func (t *tokenSource) Token(ctx context.Context) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := t.now()
	if t.accessToken != "" && now.Before(t.accessExpiresAt.Add(-tokenExpirySkew)) {
		return t.accessToken, nil
	}
	if t.refreshToken != "" && now.Before(t.refreshExpiresAt.Add(-tokenExpirySkew)) {
		if _, err := t.refreshLocked(ctx); err == nil {
			return t.accessToken, nil
		}
	}
	if _, err := t.createLocked(ctx); err != nil {
		return "", err
	}
	return t.accessToken, nil
}

func (t *tokenSource) create(ctx context.Context) (*access.TokenPair, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.createLocked(ctx)
}

func (t *tokenSource) refresh(ctx context.Context) (*access.TokenPair, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.refreshLocked(ctx)
}

func (t *tokenSource) createLocked(ctx context.Context) (*access.TokenPair, error) {
	req := t.rest.R().SetContext(ctx).
		SetHeader("Authorization", "AppSecret "+t.appSecret).
		SetHeader("X-TERMINAL-ID", t.terminalID)
	return t.exchange(req, ecommercePath+"/auth/create")
}

func (t *tokenSource) refreshLocked(ctx context.Context) (*access.TokenPair, error) {
	req := t.rest.R().SetContext(ctx).SetAuthToken(t.refreshToken)
	return t.exchange(req, ecommercePath+"/auth/refresh")
}

func (t *tokenSource) exchange(req *resty.Request, path string) (*access.TokenPair, error) {
	var out access.TokenPair
	if err := t.rest.Do(req, http.MethodGet, path, &out); err != nil {
		return nil, err
	}
	now := t.now()
	t.accessToken = out.AccessToken
	t.accessExpiresAt = now.Add(time.Duration(out.ExpiresIn) * time.Second)
	// auth/refresh may omit the refresh token; keep the one we already have in that case.
	if out.RefreshToken != "" {
		t.refreshToken = out.RefreshToken
		t.refreshExpiresAt = now.Add(time.Duration(out.RefreshExpiresIn) * time.Second)
	}
	return &out, nil
}

// CreateToken forces a fresh TokenPair via auth/create.
func (c *Client) CreateToken(ctx context.Context) (*access.TokenPair, error) { return c.tokens.create(ctx) }

// RefreshToken exchanges the cached refresh token via auth/refresh.
func (c *Client) RefreshToken(ctx context.Context) (*access.TokenPair, error) { return c.tokens.refresh(ctx) }
