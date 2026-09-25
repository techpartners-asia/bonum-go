package gateway_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	bonum "github.com/techpartners-asia/bonum-go"
)

// memStore is a TokenStore shared between clients, standing in for Redis.
type memStore struct {
	mu    sync.Mutex
	tok   bonum.StoredToken
	ok    bool
	saves int
}

func (m *memStore) Load(context.Context) (bonum.StoredToken, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.tok, m.ok, nil
}

func (m *memStore) Save(_ context.Context, t bonum.StoredToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tok, m.ok = t, true
	m.saves++
	return nil
}

// tokenGateway counts token requests. createStatus overrides auth/create's answer.
type tokenGateway struct {
	creates      atomic.Int32
	refreshes    atomic.Int32
	createStatus atomic.Int32
	url          string
}

func newTokenGateway(t *testing.T) *tokenGateway {
	t.Helper()
	g := &tokenGateway{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case ecommercePath + "/auth/create":
			g.creates.Add(1)
			if s := g.createStatus.Load(); s != 0 {
				w.WriteHeader(int(s))
				w.Write([]byte(`{"traceId":"t-429","message":"Rate-Limit: Use previous token. Do not get token too frequently,","status":429}`))
				return
			}
			w.Write([]byte(`{"tokenType":"Bearer","accessToken":"access-1","expiresIn":1800,"refreshToken":"refresh-1","refreshExpiresIn":2000,"unit":"SECONDS"}`))
		case ecommercePath + "/auth/refresh":
			g.refreshes.Add(1)
			w.Write([]byte(`{"tokenType":"Bearer","accessToken":"access-2","expiresIn":1800}`))
		case ecommercePath + "/invoices/payment-providers":
			if r.Header.Get("Authorization") == "Bearer revoked" {
				w.WriteHeader(401)
				w.Write([]byte(`{"traceId":"t-401","message":"invalid token","status":401}`))
				return
			}
			w.Write([]byte(`[{"provider":"QPAY","enabled":true}]`))
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)
	g.url = srv.URL
	return g
}

func (g *tokenGateway) client(t *testing.T, store bonum.TokenStore) *bonum.Client {
	t.Helper()
	c := bonum.New(bonum.Sandbox, "secret-123", "17171119", bonum.WithBaseURL(g.url), bonum.WithTokenStore(store))
	t.Cleanup(func() { c.Close() })
	return c
}

func call(t *testing.T, c *bonum.Client) error {
	t.Helper()
	_, err := c.Invoices.Providers(context.Background())
	return err
}

// Two replicas, or one process before and after a restart: the second one must use the
// token the first was given rather than asking auth/create again.
func TestTokenStoreSharesOneTokenAcrossClients(t *testing.T) {
	g := newTokenGateway(t)
	store := &memStore{}

	if err := call(t, g.client(t, store)); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := call(t, g.client(t, store)); err != nil {
			t.Fatal(err)
		}
	}
	if n := g.creates.Load(); n != 1 {
		t.Fatalf("auth/create called %d times across clients sharing a store, want 1", n)
	}
	if store.tok.AccessToken != "access-1" || store.tok.RefreshToken != "refresh-1" {
		t.Fatalf("stored %+v", store.tok)
	}
}

// Expired access token, live refresh token in the store: refresh, never create.
func TestTokenStoreExpiredAccessIsRefreshed(t *testing.T) {
	g := newTokenGateway(t)
	store := &memStore{ok: true, tok: bonum.StoredToken{
		AccessToken: "access-old", AccessExpiresAt: time.Now().Add(-time.Minute),
		RefreshToken: "refresh-1", RefreshExpiresAt: time.Now().Add(time.Hour),
	}}
	if err := call(t, g.client(t, store)); err != nil {
		t.Fatal(err)
	}
	if g.creates.Load() != 0 || g.refreshes.Load() != 1 {
		t.Fatalf("creates=%d refreshes=%d, want 0 and 1", g.creates.Load(), g.refreshes.Load())
	}
	if store.tok.AccessToken != "access-2" || store.tok.RefreshToken != "refresh-1" {
		t.Fatalf("stored %+v", store.tok)
	}
}

// Denied "Use previous token" while another process has just stored one: use that.
func TestTokenStoreRateLimitedCreateAdoptsTheSharedToken(t *testing.T) {
	g := newTokenGateway(t)
	g.createStatus.Store(429)
	store := &racingStore{memStore: &memStore{}, onCreate: bonum.StoredToken{
		AccessToken: "access-1", AccessExpiresAt: time.Now().Add(time.Hour),
	}}
	if err := call(t, g.client(t, store)); err != nil {
		t.Fatalf("want the shared token to be used after the 429, got %v", err)
	}
}

// racingStore is empty on the first Load and holds a token afterwards, as when another
// replica saves one between our Load and our auth/create.
type racingStore struct {
	*memStore
	loads    atomic.Int32
	onCreate bonum.StoredToken
}

func (r *racingStore) Load(ctx context.Context) (bonum.StoredToken, bool, error) {
	if r.loads.Add(1) == 1 {
		return bonum.StoredToken{}, false, nil
	}
	return r.onCreate, true, nil
}

// A token Bonum rejects must not be reused until its nominal expiry.
func TestTokenStoreRevokedTokenIsDropped(t *testing.T) {
	g := newTokenGateway(t)
	store := &memStore{ok: true, tok: bonum.StoredToken{AccessToken: "revoked", AccessExpiresAt: time.Now().Add(time.Hour)}}
	c := g.client(t, store)
	if err := call(t, c); err == nil {
		t.Fatal("want the revoked token's 401")
	}
	if store.tok.AccessToken != "" {
		t.Fatalf("revoked token still stored: %+v", store.tok)
	}
	if err := call(t, c); err != nil {
		t.Fatalf("next call should obtain a new token: %v", err)
	}
	if g.creates.Load() != 1 {
		t.Fatalf("creates = %d, want 1", g.creates.Load())
	}
}
