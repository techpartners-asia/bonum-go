package bonum

import (
	"github.com/techpartners-asia/bonum-go/gateway/application"
	"github.com/techpartners-asia/bonum-go/gateway/domain/card"
)

// CardService is the Card Token aggregate's use cases: tokenization and charges against
// stored cards.
type CardService = application.Cards

// ErrDeclined marks a Purchase the bank refused. Match with errors.Is; the *DeclinedError
// carrying Bonum's Purchase record is available through errors.As.
var ErrDeclined = card.ErrDeclined // card purchase refused by the bank

// DeclinedError is an APIError whose body carried a FAILED Purchase.
type DeclinedError = card.DeclinedError

// PurchaseStatus is the outcome of a Purchase.
type PurchaseStatus = card.PurchaseStatus

type (
	// TokenizePayment charges the card while it is being tokenized. Defaults to 0.01 MNT if omitted.
	TokenizePayment = card.TokenizePayment
	// TokenizeSubscription enrols the new Card Token in a Payment Plan in the same flow.
	TokenizeSubscription = card.TokenizeSubscription
	TokenizeInput        = card.TokenizeInput
	// Tokenization is a hosted card-entry session. The Card Token arrives as a CardTokenEvent.
	Tokenization  = card.Tokenization
	PurchaseInput = card.PurchaseInput
	// Purchase is a charge against a Card Token.
	Purchase = card.Purchase
)

const (
	PurchaseSuccess = card.PurchaseSuccess
	PurchaseFailed  = card.PurchaseFailed
	PurchaseQueued  = card.PurchaseQueued // Result arrives later as a TokenPaymentEvent
)
