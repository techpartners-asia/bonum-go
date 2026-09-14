package domain_test

import (
	"errors"
	"testing"

	"github.com/techpartners-asia/bonum-go/gateway/domain"
	"github.com/techpartners-asia/bonum-go/gateway/domain/card"
)

func TestTokenizeInputValidate(t *testing.T) {
	if err := (card.TokenizeInput{Callback: "https://m/cb", TransactionID: "T1"}).Validate(); err != nil {
		t.Fatal(err)
	}
	for field, in := range map[string]card.TokenizeInput{
		"Callback":      {TransactionID: "T1"},
		"TransactionID": {Callback: "https://m/cb"},
	} {
		var v *domain.ValidationError
		if err := in.Validate(); !errors.As(err, &v) || v.Field != field {
			t.Fatalf("%s: got %v", field, err)
		}
	}
}

func TestPurchaseInputValidate(t *testing.T) {
	if err := (card.PurchaseInput{Amount: 1, Currency: "MNT", TransactionID: "T1"}).Validate(); err != nil {
		t.Fatal(err)
	}
	for field, in := range map[string]card.PurchaseInput{
		"Amount":        {Currency: "MNT", TransactionID: "T1"},
		"Currency":      {Amount: 1, TransactionID: "T1"},
		"TransactionID": {Amount: 1, Currency: "MNT"},
	} {
		var v *domain.ValidationError
		if err := in.Validate(); !errors.As(err, &v) || v.Field != field {
			t.Fatalf("%s: got %v", field, err)
		}
	}
}

func TestDeclinedErrorMatchesDeclinedAndInvalidInput(t *testing.T) {
	err := &card.DeclinedError{
		APIError: &domain.APIError{StatusCode: 400, TraceID: "t-d", Message: "Insufficient funds"},
		Purchase: card.Purchase{ID: 9, Status: card.PurchaseFailed},
	}
	if !errors.Is(err, card.ErrDeclined) || !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatal("decline must satisfy both ErrDeclined and ErrInvalidInput")
	}
	var api *domain.APIError
	if !errors.As(err, &api) || api.TraceID != "t-d" {
		t.Fatal("APIError must be reachable through errors.As")
	}
}
