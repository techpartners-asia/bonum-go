package domain_test

import (
	"errors"
	"testing"

	"github.com/techpartners-asia/bonum-go/internal/gateway/domain"
	"github.com/techpartners-asia/bonum-go/internal/gateway/domain/card"
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

func TestTokenizeInputValidatesSubscription(t *testing.T) {
	base := card.TokenizeInput{Callback: "https://m/cb", TransactionID: "T1"}

	valid := []card.TokenizeSubscription{
		{PlanID: 30, CycleValue: "1"},
		{PlanID: 30, CycleValue: "366"},
		{PlanID: 30, PayNow: true}, // CycleValue ignored when PayNow
	}
	for i, sub := range valid {
		in := base
		in.Subscription = &sub
		if err := in.Validate(); err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
	}

	for field, sub := range map[string]card.TokenizeSubscription{
		"Subscription.PlanID":     {CycleValue: "1"},
		"Subscription.CycleValue": {PlanID: 30, CycleValue: "0"},
	} {
		in := base
		in.Subscription = &sub
		var v *domain.ValidationError
		if err := in.Validate(); !errors.As(err, &v) || v.Field != field {
			t.Fatalf("%s: got %v", field, err)
		}
	}

	for _, bad := range []string{"367", "", "not-a-number"} {
		in := base
		in.Subscription = &card.TokenizeSubscription{PlanID: 30, CycleValue: bad}
		var v *domain.ValidationError
		if err := in.Validate(); !errors.As(err, &v) || v.Field != "Subscription.CycleValue" {
			t.Fatalf("CycleValue=%q: got %v", bad, err)
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
