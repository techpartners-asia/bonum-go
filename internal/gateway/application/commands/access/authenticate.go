// Package access holds the Access aggregate's commands: forcing a fresh token or refresh.
package access

import (
	"context"

	"github.com/techpartners-asia/bonum-go/internal/gateway/domain/access"
	"github.com/techpartners-asia/bonum-go/internal/gateway/ports"
)

// AuthenticateHandler forces a fresh TokenPair via auth/create. The endpoint is rate
// limited; do not call it in a loop.
type AuthenticateHandler struct{ api ports.AccessAPI }

func NewAuthenticateHandler(api ports.AccessAPI) *AuthenticateHandler {
	return &AuthenticateHandler{api: api}
}

// Handle takes no input: Authenticate always re-derives credentials from the Terminal's
// AppSecret held by the adapter.
func (h *AuthenticateHandler) Handle(ctx context.Context) (*access.TokenPair, error) {
	return h.api.CreateToken(ctx)
}
