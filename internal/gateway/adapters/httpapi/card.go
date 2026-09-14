package httpapi

import (
	"context"
	"net/http"

	"github.com/techpartners-asia/bonum-go/internal/gateway/domain/card"
	"github.com/techpartners-asia/bonum-go/internal/gateway/ports"
)

var _ ports.CardAPI = (*Client)(nil)

func (c *Client) Tokenize(ctx context.Context, in card.TokenizeInput) (*card.Tokenization, error) {
	return call[card.Tokenization](ctx, c, http.MethodPost, mpayPath+"/cards/tokenize/request", body(in))
}

func (c *Client) Purchase(ctx context.Context, cardTok string, in card.PurchaseInput) (*card.Purchase, error) {
	p, err := callEnveloped[card.Purchase](ctx, c, http.MethodPost, mpayPath+"/transaction/purchase", cardToken(cardTok), body(in))
	if err != nil {
		return nil, asDeclined(err)
	}
	return p, nil
}

func (c *Client) Reverse(ctx context.Context, cardTok, transactionID string) error {
	return callAction(ctx, c, http.MethodDelete, mpayPath+"/transaction/reverse/{transactionId}",
		cardToken(cardTok), pathParam("transactionId", transactionID))
}
