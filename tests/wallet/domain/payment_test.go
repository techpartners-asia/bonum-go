package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/techpartners-asia/bonum-go/internal/wallet/domain"
	"github.com/techpartners-asia/bonum-go/internal/wallet/domain/payment"
)

func appleToken() payment.ApplePayToken {
	return payment.ApplePayToken{
		PaymentData:           payment.ApplePaymentData{Data: "aGVsbG8=", Signature: "MIAG", Version: "EC_v1"},
		TransactionIdentifier: "9AFDA47D",
	}
}

func TestProcessApplePayInputValidate(t *testing.T) {
	if err := (payment.ProcessApplePayInput{OrderID: "o", Amount: 150.50, BranchID: "b", Token: appleToken()}).Validate(); err != nil {
		t.Fatal(err)
	}
	for field, in := range map[string]payment.ProcessApplePayInput{
		"OrderID":  {Token: appleToken()},
		"Amount":   {OrderID: "o", Amount: 0.001, Token: appleToken()},
		"BranchID": {OrderID: "o", BranchID: strings.Repeat("b", 65), Token: appleToken()},
		"Token":    {OrderID: "o"},
	} {
		var v *domain.ValidationError
		if err := in.Validate(); !errors.As(err, &v) || v.Field != field || !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("%s: got %v", field, err)
		}
	}
	long := payment.ProcessApplePayInput{OrderID: strings.Repeat("x", 129), Token: appleToken()}
	if err := long.Validate(); err == nil {
		t.Fatal("129-char order id must fail")
	}
	dec := payment.ProcessApplePayInput{OrderID: "o", Amount: 1.234, Token: appleToken()}
	if err := dec.Validate(); err == nil {
		t.Fatal("3 decimals must fail")
	}
}

func TestProcessGooglePayInputValidate(t *testing.T) {
	if err := (payment.ProcessGooglePayInput{OrderID: "o", Token: "t", CurrencyCode: payment.MNT}).Validate(); err != nil {
		t.Fatal(err)
	}
	for field, in := range map[string]payment.ProcessGooglePayInput{
		"Token":        {OrderID: "o", CurrencyCode: payment.MNT},
		"CurrencyCode": {OrderID: "o", Token: "t", CurrencyCode: "GBP"},
	} {
		var v *domain.ValidationError
		if err := in.Validate(); !errors.As(err, &v) || v.Field != field {
			t.Fatalf("%s: got %v", field, err)
		}
	}
}

func TestAPIErrorMapsStatus(t *testing.T) {
	err := &domain.APIError{StatusCode: 401, Message: "Invalid key", ErrorText: "Unauthorized"}
	if !errors.Is(err, domain.ErrUnauthorized) || err.Error() != "bonum wallet: 401 Invalid key" {
		t.Fatalf("unexpected %v", err)
	}
	if !errors.Is(&domain.APIError{StatusCode: 429}, domain.ErrRateLimited) {
		t.Fatal("429")
	}
}
