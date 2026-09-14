package httpapi

import (
	"encoding/json"
	"errors"

	"github.com/techpartners-asia/bonum-go/gateway/domain"
	"github.com/techpartners-asia/bonum-go/gateway/domain/card"
)

type errorBody struct {
	TraceID string `json:"traceId"`
	Message string `json:"message"`
}

// decodeError turns a non-2xx response into *domain.APIError.
func decodeError(status int, body string) error {
	e := &domain.APIError{StatusCode: status, Body: body}
	var eb errorBody
	if json.Unmarshal([]byte(body), &eb) == nil {
		e.TraceID = eb.TraceID
		e.Message = eb.Message
	}
	return e
}

// asDeclined upgrades a 400 whose body is a FAILED Purchase envelope into *card.DeclinedError.
func asDeclined(err error) error {
	var api *domain.APIError
	if !errors.As(err, &api) || api.StatusCode != 400 {
		return err
	}
	var env envelope[card.Purchase]
	if json.Unmarshal([]byte(api.Body), &env) != nil || env.Data.Status != card.PurchaseFailed {
		return err
	}
	return &card.DeclinedError{APIError: api, Purchase: env.Data}
}
