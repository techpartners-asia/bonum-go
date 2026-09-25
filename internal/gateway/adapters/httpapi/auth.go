package httpapi

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/techpartners-asia/bonum-go/internal/gateway/domain"
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
	// store, when set, shares the token across processes; see access.TokenStore.
	store access.TokenStore

	mu               sync.Mutex
	accessToken      string
	refreshToken     string
	accessExpiresAt  time.Time
	refreshExpiresAt time.Time
}

func newTokenSource(r *rest.Client, appSecret, terminalID string) *tokenSource {
	return &tokenSource{rest: r, appSecret: appSecret, terminalID: terminalID, now: time.Now}
}

// Token returns a usable access token: the one held in memory, else the shared one in the
// store, else a refreshed one, else a new one from auth/create - in that order, so a token
// is used until it expires and auth/create is the last resort.
func (t *tokenSource) Token(ctx context.Context) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := t.now()
	if t.current().AccessValidAt(now, tokenExpirySkew) {
		return t.accessToken, nil
	}
	if t.adoptStored(ctx, now) {
		return t.accessToken, nil
	}
	if t.current().RefreshValidAt(now, tokenExpirySkew) {
		if _, err := t.refreshLocked(ctx); err == nil {
			return t.accessToken, nil
		}
	}
	if _, err := t.createLocked(ctx); err != nil {
		// Refused for asking too often: another process sharing the store may have
		// just been given the token we were denied. Use it if so.
		if errors.Is(err, domain.ErrRateLimited) && t.adoptStored(ctx, t.now()) {
			return t.accessToken, nil
		}
		return "", err
	}
	return t.accessToken, nil
}

// Invalidate drops token after Bonum rejected it (401/403), in memory and in the store if
// the store still holds it, so the next call obtains a new one instead of reusing a dead
// one until its nominal expiry.
func (t *tokenSource) Invalidate(ctx context.Context, token string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if token == "" || token != t.accessToken {
		return
	}
	t.accessToken, t.accessExpiresAt = "", time.Time{}
	if t.store == nil {
		return
	}
	if cur, ok, err := t.store.Load(ctx); err == nil && ok && cur.AccessToken == token {
		cur.AccessToken, cur.AccessExpiresAt = "", time.Time{}
		_ = t.store.Save(ctx, cur)
	}
}

func (t *tokenSource) current() access.StoredToken {
	return access.StoredToken{
		AccessToken: t.accessToken, AccessExpiresAt: t.accessExpiresAt,
		RefreshToken: t.refreshToken, RefreshExpiresAt: t.refreshExpiresAt,
	}
}

// adoptStored takes the store's token when its access token is still valid. Otherwise it
// still picks up a usable refresh token when we hold none, so a refresh can stand in for a
// create.
func (t *tokenSource) adoptStored(ctx context.Context, now time.Time) bool {
	if t.store == nil {
		return false
	}
	cur, ok, err := t.store.Load(ctx)
	if err != nil || !ok {
		return false
	}
	if cur.AccessValidAt(now, tokenExpirySkew) {
		t.accessToken, t.accessExpiresAt = cur.AccessToken, cur.AccessExpiresAt
		t.refreshToken, t.refreshExpiresAt = cur.RefreshToken, cur.RefreshExpiresAt
		return true
	}
	if !t.current().RefreshValidAt(now, tokenExpirySkew) && cur.RefreshValidAt(now, tokenExpirySkew) {
		t.refreshToken, t.refreshExpiresAt = cur.RefreshToken, cur.RefreshExpiresAt
	}
	return false
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
	if t.store != nil {
		// Best effort: a failed save only costs the other processes a token request.
		_ = t.store.Save(req.Context(), t.current())
	}
	return &out, nil
}

// CreateToken forces a fresh TokenPair via auth/create.
func (c *Client) CreateToken(ctx context.Context) (*access.TokenPair, error) {
	return c.tokens.create(ctx)
}

// RefreshToken exchanges the cached refresh token via auth/refresh.
func (c *Client) RefreshToken(ctx context.Context) (*access.TokenPair, error) {
	return c.tokens.refresh(ctx)
}
