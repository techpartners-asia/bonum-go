package bonum

import (
	"net/http"

	"github.com/techpartners-asia/bonum-go/types"
)

const cardTokenHeader = "X-CARD-TOKEN"

// CreateCardToken starts a card tokenization flow. Redirect the customer to FollowUpLink;
// the token arrives on the merchant webhook as a CARD-TOKEN message.
func (c *Client) CreateCardToken(input types.CreateCardTokenInput) (*types.CreateCardTokenResponse, error) {
	req, err := c.authedRequest()
	if err != nil {
		return nil, err
	}
	var out types.CreateCardTokenResponse
	if err := c.do(req.SetBody(input), http.MethodPost, mpayPath+"/cards/tokenize/request", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Purchase charges a stored card token. Under load Bonum may answer 201 with Status QUEUED;
// the final result then arrives as a TOKEN-PAYMENT webhook. A declined card comes back as
// a *Error with StatusCode 400 whose Body still contains the PurchaseResponse envelope.
func (c *Client) Purchase(cardToken string, input types.PurchaseInput) (*types.PurchaseResponse, error) {
	req, err := c.authedRequest()
	if err != nil {
		return nil, err
	}
	var out types.PurchaseResponse
	if err := c.do(req.SetHeader(cardTokenHeader, cardToken).SetBody(input), http.MethodPost, mpayPath+"/transaction/purchase", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// RollbackPurchase attempts to reverse a token purchase identified by the merchant transactionId.
func (c *Client) RollbackPurchase(cardToken, transactionID string) (*types.RollbackPurchaseResponse, error) {
	req, err := c.authedRequest()
	if err != nil {
		return nil, err
	}
	var out types.RollbackPurchaseResponse
	if err := c.do(req.SetHeader(cardTokenHeader, cardToken).SetPathParam("transactionId", transactionID),
		http.MethodDelete, mpayPath+"/transaction/reverse/{transactionId}", &out); err != nil {
		return nil, err
	}
	return &out, nil
}
