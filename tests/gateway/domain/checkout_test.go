package domain_test

import (
	"errors"
	"testing"

	"github.com/techpartners-asia/bonum-go/internal/gateway/domain"
	"github.com/techpartners-asia/bonum-go/internal/gateway/domain/checkout"
)

func TestCreateInvoiceInputValidate(t *testing.T) {
	valid := checkout.CreateInvoiceInput{Amount: 100, Callback: "https://m/cb", TransactionID: "T1", ExpiresIn: 600}
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	cases := map[string]checkout.CreateInvoiceInput{
		"Amount":        {Callback: "https://m/cb", TransactionID: "T1", ExpiresIn: 600},
		"Callback":      {Amount: 100, TransactionID: "T1", ExpiresIn: 600},
		"TransactionID": {Amount: 100, Callback: "https://m/cb", ExpiresIn: 600},
		"ExpiresIn":     {Amount: 100, Callback: "https://m/cb", TransactionID: "T1"},
	}
	for field, in := range cases {
		err := in.Validate()
		var v *domain.ValidationError
		if !errors.As(err, &v) || v.Field != field || !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("%s: got %v", field, err)
		}
	}
}
