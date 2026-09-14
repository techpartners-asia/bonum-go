// Package access is the Access aggregate: the Terminal's bearer credentials.
package access

// TokenPair is returned by auth/create and auth/refresh.
type TokenPair struct {
	TokenType        string `json:"tokenType"` // "Bearer"
	AccessToken      string `json:"accessToken"`
	ExpiresIn        int64  `json:"expiresIn"` // access token ttl, in Unit
	RefreshToken     string `json:"refreshToken"`
	RefreshExpiresIn int64  `json:"refreshExpiresIn"` // refresh token ttl, in Unit
	Unit             string `json:"unit"`             // "SECONDS"
}
