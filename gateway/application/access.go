// Package application holds the Gateway facades: one struct per aggregate exposing the same
// public methods as before, each delegating to a Command/Query Handler in
// application/commands/<aggregate> or application/queries/<aggregate>. This is the CQRS-lite
// split: one Handler per use case, kept behind a facade so callers (ultimately the bonum
// package) see one object per aggregate rather than one per use case.
package application

import (
	"context"

	accesscmd "github.com/techpartners-asia/bonum-go/gateway/application/commands/access"
	"github.com/techpartners-asia/bonum-go/gateway/domain/access"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// Access forces token creation or refresh. Normally unnecessary: every call obtains a token
// on demand inside the adapter.
type Access struct {
	authenticate *accesscmd.AuthenticateHandler
	refresh      *accesscmd.RefreshHandler
}

func NewAccess(api ports.AccessAPI) *Access {
	return &Access{
		authenticate: accesscmd.NewAuthenticateHandler(api),
		refresh:      accesscmd.NewRefreshHandler(api),
	}
}

// Authenticate forces a fresh TokenPair via auth/create. The endpoint is rate limited.
func (s *Access) Authenticate(ctx context.Context) (*access.TokenPair, error) {
	return s.authenticate.Handle(ctx)
}

// Refresh exchanges the cached refresh token for a new access token via auth/refresh.
func (s *Access) Refresh(ctx context.Context) (*access.TokenPair, error) {
	return s.refresh.Handle(ctx)
}
