// Package access is the Access aggregate: the Terminal's bearer credentials.
package access

import (
	"context"
	"time"
)

// TokenPair is returned by auth/create and auth/refresh.
type TokenPair struct {
	TokenType        string `json:"tokenType"` // "Bearer"
	AccessToken      string `json:"accessToken"`
	ExpiresIn        int64  `json:"expiresIn"` // access token ttl, in Unit
	RefreshToken     string `json:"refreshToken"`
	RefreshExpiresIn int64  `json:"refreshExpiresIn"` // refresh token ttl, in Unit
	Unit             string `json:"unit"`             // "SECONDS"
}

// StoredToken is the Terminal's current credentials with absolute expiry times, the form
// a TokenStore keeps them in.
type StoredToken struct {
	AccessToken      string    `json:"accessToken"`
	AccessExpiresAt  time.Time `json:"accessExpiresAt"`
	RefreshToken     string    `json:"refreshToken,omitempty"`
	RefreshExpiresAt time.Time `json:"refreshExpiresAt"`
}

// AccessValidAt reports whether the access token is usable at t, skew early.
func (s StoredToken) AccessValidAt(t time.Time, skew time.Duration) bool {
	return s.AccessToken != "" && t.Before(s.AccessExpiresAt.Add(-skew))
}

// RefreshValidAt reports whether the refresh token is usable at t, skew early.
func (s StoredToken) RefreshValidAt(t time.Time, skew time.Duration) bool {
	return s.RefreshToken != "" && t.Before(s.RefreshExpiresAt.Add(-skew))
}

// TokenStore keeps the Terminal's token somewhere every process can see it (Redis, a
// database row). Bonum rate-limits auth/create per Terminal and answers "Use previous
// token" when asked too often, so every replica, restart and deploy that asks for its own
// token competes for that limit. With a store the client reuses the shared token until it
// expires, only then refreshes or creates one, and saves it for the others.
//
// Load returns ok=false when nothing is stored. Errors from either method never fail a
// call: the client falls back to its in-memory token and to Bonum.
type TokenStore interface {
	Load(ctx context.Context) (token StoredToken, ok bool, err error)
	Save(ctx context.Context, token StoredToken) error
}
