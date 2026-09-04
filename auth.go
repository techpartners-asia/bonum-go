package bonum

import (
	"net/http"
	"time"

	"github.com/techpartners-asia/bonum-go/types"
)

// Authenticate forces a fresh token pair via auth/create and caches it.
// You normally never need this: every call obtains a token on demand. The endpoint is
// rate limited, so avoid calling it in a loop.
func (c *Client) Authenticate() (*types.AuthResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.createTokenLocked()
}

// Refresh exchanges the cached refresh token for a new access token via auth/refresh.
func (c *Client) Refresh() (*types.AuthResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.refreshTokenLocked()
}

// token returns a usable access token, refreshing or re-authenticating as needed.
func (c *Client) token() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	if c.accessToken != "" && now.Before(c.accessExpiresAt.Add(-tokenExpirySkew)) {
		return c.accessToken, nil
	}
	if c.refreshToken != "" && now.Before(c.refreshExpiresAt.Add(-tokenExpirySkew)) {
		if _, err := c.refreshTokenLocked(); err == nil {
			return c.accessToken, nil
		}
	}
	if _, err := c.createTokenLocked(); err != nil {
		return "", err
	}
	return c.accessToken, nil
}

func (c *Client) createTokenLocked() (*types.AuthResponse, error) {
	var out types.AuthResponse
	req := c.http.R().
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", "AppSecret "+c.appSecret).
		SetHeader("X-TERMINAL-ID", c.terminalID)
	if err := c.do(req, http.MethodGet, ecommercePath+"/auth/create", &out); err != nil {
		return nil, err
	}
	c.storeTokensLocked(&out)
	return &out, nil
}

func (c *Client) refreshTokenLocked() (*types.AuthResponse, error) {
	var out types.AuthResponse
	req := c.http.R().
		SetHeader("Accept", "application/json").
		SetAuthToken(c.refreshToken)
	if err := c.do(req, http.MethodGet, ecommercePath+"/auth/refresh", &out); err != nil {
		return nil, err
	}
	c.storeTokensLocked(&out)
	return &out, nil
}

func (c *Client) storeTokensLocked(a *types.AuthResponse) {
	now := time.Now()
	c.accessToken = a.AccessToken
	c.accessExpiresAt = now.Add(time.Duration(a.ExpiresIn) * time.Second)
	// auth/refresh may omit the refresh token; keep the one we already have in that case.
	if a.RefreshToken != "" {
		c.refreshToken = a.RefreshToken
		c.refreshExpiresAt = now.Add(time.Duration(a.RefreshExpiresIn) * time.Second)
	}
}
