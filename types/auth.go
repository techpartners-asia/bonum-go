package types

// AuthResponse is returned by both auth/create and auth/refresh.
type AuthResponse struct {
	TokenType        string `json:"tokenType"` // "Bearer"
	AccessToken      string `json:"accessToken"`
	ExpiresIn        int64  `json:"expiresIn"` // access token ttl, in Unit
	RefreshToken     string `json:"refreshToken"`
	RefreshExpiresIn int64  `json:"refreshExpiresIn"` // refresh token ttl, in Unit
	Unit             string `json:"unit"`             // "SECONDS"
}

// ErrorResponse is the body Bonum returns on non-2xx responses.
type ErrorResponse struct {
	TraceID   string  `json:"traceId"`
	ErrorCode *string `json:"errorCode"`
	Error     *string `json:"error"`
	Message   string  `json:"message"`
	Detail    *string `json:"detail"`
	Status    int     `json:"status"`
}
