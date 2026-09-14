package httpapi

import (
	"encoding/json"

	"github.com/techpartners-asia/bonum-go/wallet/domain"
)

type errorBody struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Error      string `json:"error"`
}

// decodeError turns a non-2xx response into *domain.APIError.
func decodeError(status int, body string) error {
	e := &domain.APIError{StatusCode: status, Body: body}
	var eb errorBody
	if json.Unmarshal([]byte(body), &eb) == nil {
		e.Message = eb.Message
		e.ErrorText = eb.Error
	}
	return e
}
