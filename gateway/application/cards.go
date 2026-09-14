package application

import (
	"context"

	cardcmd "github.com/techpartners-asia/bonum-go/gateway/application/commands/card"
	"github.com/techpartners-asia/bonum-go/gateway/domain/card"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// Cards is the Card Token aggregate's use cases: tokenization and charges against stored cards.
type Cards struct {
	tokenize *cardcmd.TokenizeHandler
	purchase *cardcmd.PurchaseHandler
	reverse  *cardcmd.ReverseHandler
}

func NewCards(api ports.CardAPI) *Cards {
	return &Cards{
		tokenize: cardcmd.NewTokenizeHandler(api),
		purchase: cardcmd.NewPurchaseHandler(api),
		reverse:  cardcmd.NewReverseHandler(api),
	}
}

// Tokenize starts a card tokenization flow. Redirect the customer to FollowUpLink.
func (s *Cards) Tokenize(ctx context.Context, in card.TokenizeInput) (*card.Tokenization, error) {
	return s.tokenize.Handle(ctx, in)
}

// Purchase charges a Card Token. Under load Bonum may answer with Status QUEUED; the final
// result then arrives as a TokenPaymentEvent. A bank refusal is returned as *card.DeclinedError.
func (s *Cards) Purchase(ctx context.Context, cardToken string, in card.PurchaseInput) (*card.Purchase, error) {
	return s.purchase.Handle(ctx, cardcmd.PurchaseCommand{CardToken: cardToken, PurchaseInput: in})
}

// Reverse rolls back a Purchase identified by the merchant TransactionID.
func (s *Cards) Reverse(ctx context.Context, cardToken, transactionID string) error {
	return s.reverse.Handle(ctx, cardcmd.ReverseCommand{CardToken: cardToken, TransactionID: transactionID})
}
