package domain_test

import (
	"errors"
	"testing"

	"github.com/techpartners-asia/bonum-go/gateway/domain"
	"github.com/techpartners-asia/bonum-go/gateway/domain/subscription"
)

func TestSubscribeInputValidate(t *testing.T) {
	ok := []subscription.SubscribeInput{
		{PlanID: 30, CycleValue: 1},
		{PlanID: 30, CycleValue: 366},
		{PlanID: 30, PayNow: true}, // CycleValue ignored when PayNow
	}
	for i, in := range ok {
		if err := in.Validate(); err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
	}
	for field, in := range map[string]subscription.SubscribeInput{
		"PlanID":     {CycleValue: 5},
		"CycleValue": {PlanID: 30, CycleValue: 0},
	} {
		var v *domain.ValidationError
		if err := in.Validate(); !errors.As(err, &v) || v.Field != field {
			t.Fatalf("%s: got %v", field, err)
		}
	}
	var v *domain.ValidationError
	if err := (subscription.SubscribeInput{PlanID: 30, CycleValue: 367}).Validate(); !errors.As(err, &v) || v.Field != "CycleValue" {
		t.Fatalf("367: got %v", err)
	}
}

func TestChangeCardInputValidate(t *testing.T) {
	if err := (subscription.ChangeCardInput{Callback: "https://m/cb", TransactionID: "T1"}).Validate(); err != nil {
		t.Fatal(err)
	}
	for field, in := range map[string]subscription.ChangeCardInput{
		"Callback":      {TransactionID: "T1"},
		"TransactionID": {Callback: "https://m/cb"},
	} {
		var v *domain.ValidationError
		if err := in.Validate(); !errors.As(err, &v) || v.Field != field {
			t.Fatalf("%s: got %v", field, err)
		}
	}
}
