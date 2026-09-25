package bonum

import "github.com/techpartners-asia/bonum-go/internal/gateway/domain/access"

// TokenPair is returned by Authenticate and Refresh.
type TokenPair = access.TokenPair

// TokenStore shares the Terminal's token across processes so it is reused until it
// expires; see WithTokenStore.
type TokenStore = access.TokenStore

// StoredToken is what a TokenStore keeps.
type StoredToken = access.StoredToken
