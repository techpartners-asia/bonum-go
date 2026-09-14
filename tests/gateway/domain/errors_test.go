package domain_test

import (
	"errors"
	"testing"

	"github.com/techpartners-asia/bonum-go/gateway/domain"
)

func TestAPIErrorMapsStatusToSentinels(t *testing.T) {
	cases := map[int]error{400: domain.ErrInvalidInput, 401: domain.ErrUnauthorized, 403: domain.ErrUnauthorized, 404: domain.ErrNotFound, 429: domain.ErrRateLimited}
	for status, want := range cases {
		err := &domain.APIError{StatusCode: status, TraceID: "t", Message: "m"}
		if !errors.Is(err, want) {
			t.Fatalf("%d: want %v", status, want)
		}
	}
	if errors.Is(&domain.APIError{StatusCode: 500}, domain.ErrInvalidInput) {
		t.Fatal("500 must not match ErrInvalidInput")
	}
	if got := (&domain.APIError{StatusCode: 404, TraceID: "t-1", Message: "gone"}).Error(); got != "bonum: 404 gone (trace t-1)" {
		t.Fatalf("Error() = %q", got)
	}
}

func TestInvalidIsValidationError(t *testing.T) {
	err := domain.Invalid("Amount", "must be positive")
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatal("want ErrInvalidInput")
	}
	var v *domain.ValidationError
	if !errors.As(err, &v) || v.Field != "Amount" || err.Error() != "bonum: invalid Amount: must be positive" {
		t.Fatalf("unexpected %#v", err)
	}
}
