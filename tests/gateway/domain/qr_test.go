package domain_test

import (
	"errors"
	"testing"

	"github.com/techpartners-asia/bonum-go/gateway/domain"
	"github.com/techpartners-asia/bonum-go/gateway/domain/qr"
)

func TestCreateQRInputValidate(t *testing.T) {
	if err := (qr.CreateQRInput{Amount: 1, TransactionID: "T1", ExpiresIn: 60}).Validate(); err != nil {
		t.Fatal(err)
	}
	for field, in := range map[string]qr.CreateQRInput{
		"Amount":        {TransactionID: "T1", ExpiresIn: 60},
		"TransactionID": {Amount: 1, ExpiresIn: 60},
		"ExpiresIn":     {Amount: 1, TransactionID: "T1"},
	} {
		var v *domain.ValidationError
		if err := in.Validate(); !errors.As(err, &v) || v.Field != field {
			t.Fatalf("%s: got %v", field, err)
		}
	}
}

func TestPayQRInputValidate(t *testing.T) {
	if err := (qr.PayQRInput{QrCode: "0002", TransactionID: "T1"}).Validate(); err != nil {
		t.Fatal(err)
	}
	for field, in := range map[string]qr.PayQRInput{
		"QrCode":        {TransactionID: "T1"},
		"TransactionID": {QrCode: "0002"},
	} {
		var v *domain.ValidationError
		if err := in.Validate(); !errors.As(err, &v) || v.Field != field {
			t.Fatalf("%s: got %v", field, err)
		}
	}
}
