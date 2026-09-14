package access

import (
	"context"

	"github.com/techpartners-asia/bonum-go/gateway/domain/access"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// RefreshHandler exchanges the cached refresh token for a new access token via auth/refresh.
type RefreshHandler struct{ api ports.AccessAPI }

func NewRefreshHandler(api ports.AccessAPI) *RefreshHandler { return &RefreshHandler{api: api} }

func (h *RefreshHandler) Handle(ctx context.Context) (*access.TokenPair, error) {
	return h.api.RefreshToken(ctx)
}
