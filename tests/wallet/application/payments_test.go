package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/techpartners-asia/bonum-go/internal/wallet/application"
	"github.com/techpartners-asia/bonum-go/internal/wallet/domain"
	"github.com/techpartners-asia/bonum-go/internal/wallet/domain/payment"
)

var ctx = context.Background()

type fakeAPI struct {
	calls   []string
	args    []any
	timeout time.Duration
	err     error
}

func (f *fakeAPI) record(name string, args ...any) { f.calls = append(f.calls, name); f.args = args }

func (f *fakeAPI) ProcessApplePay(_ context.Context, in payment.ProcessApplePayInput) (*payment.ProcessResponse, error) {
	f.record("ProcessApplePay", in)
	return &payment.ProcessResponse{PaymentID: "p1", Status: payment.StatusPending}, f.err
}
func (f *fakeAPI) ProcessGooglePay(_ context.Context, in payment.ProcessGooglePayInput) (*payment.ProcessResponse, error) {
	f.record("ProcessGooglePay", in)
	return &payment.ProcessResponse{PaymentID: "p2", Status: payment.StatusPending}, f.err
}
func (f *fakeAPI) GetPayment(_ context.Context, id string) (*payment.Payment, error) {
	f.record("GetPayment", id)
	return &payment.Payment{PaymentID: id}, f.err
}
func (f *fakeAPI) LookupByOrderID(_ context.Context, orderID string) (*payment.Payment, error) {
	f.record("LookupByOrderID", orderID)
	return &payment.Payment{OrderID: orderID}, f.err
}
func (f *fakeAPI) AwaitPayment(_ context.Context, id string, timeout time.Duration) (*payment.AwaitResult, error) {
	f.record("AwaitPayment", id)
	f.timeout = timeout
	return &payment.AwaitResult{PaymentID: id, Status: payment.StatusAuthorized}, f.err
}
func (f *fakeAPI) AwaitURL(_ context.Context, url string, timeout time.Duration) (*payment.AwaitResult, error) {
	f.record("AwaitURL", url)
	f.timeout = timeout
	return &payment.AwaitResult{Status: payment.StatusAuthorized}, f.err
}

func appleToken() payment.ApplePayToken {
	return payment.ApplePayToken{PaymentData: payment.ApplePaymentData{Data: "d"}, TransactionIdentifier: "t"}
}

func TestProcessValidatesBeforePort(t *testing.T) {
	f := &fakeAPI{}
	s := application.NewPayments(f)
	if _, err := s.ProcessApplePay(ctx, payment.ProcessApplePayInput{}); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("apple: %v", err)
	}
	if _, err := s.ProcessGooglePay(ctx, payment.ProcessGooglePayInput{OrderID: "o"}); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("google: %v", err)
	}
	if len(f.calls) != 0 {
		t.Fatal("invalid input must not reach the port")
	}
	in := payment.ProcessApplePayInput{OrderID: "o", Token: appleToken()}
	if res, err := s.ProcessApplePay(ctx, in); err != nil || res.PaymentID != "p1" || f.args[0] != in {
		t.Fatalf("forwarding: %v %v", res, err)
	}
	g := payment.ProcessGooglePayInput{OrderID: "o", Token: "t", CurrencyCode: payment.USD}
	if res, err := s.ProcessGooglePay(ctx, g); err != nil || res.PaymentID != "p2" || f.args[0] != g {
		t.Fatalf("forwarding: %v %v", res, err)
	}
}

func TestIDsAreRequired(t *testing.T) {
	f := &fakeAPI{}
	s := application.NewPayments(f)
	if _, err := s.GetPayment(ctx, ""); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatal(err)
	}
	if _, err := s.LookupByOrderID(ctx, ""); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatal(err)
	}
	if _, err := s.AwaitPayment(ctx, "", time.Second); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatal(err)
	}
	if _, err := s.AwaitURL(ctx, "", time.Second); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatal(err)
	}
	if len(f.calls) != 0 {
		t.Fatal("empty ids must not reach the port")
	}
	if p, err := s.GetPayment(ctx, "p1"); err != nil || p.PaymentID != "p1" {
		t.Fatal(err)
	}
	if p, err := s.LookupByOrderID(ctx, "o1"); err != nil || p.OrderID != "o1" {
		t.Fatal(err)
	}
}

func TestAwaitTimeoutIsClamped(t *testing.T) {
	f := &fakeAPI{}
	s := application.NewPayments(f)
	cases := map[time.Duration]time.Duration{
		0:                time.Duration(0),
		-time.Second:     time.Duration(0),
		2 * time.Second:  2 * time.Second,
		time.Minute:      payment.MaxAwaitTimeout,
		28 * time.Second: payment.MaxAwaitTimeout,
	}
	for in, want := range cases {
		if _, err := s.AwaitPayment(ctx, "p1", in); err != nil || f.timeout != want {
			t.Fatalf("AwaitPayment(%v): timeout %v, want %v (err %v)", in, f.timeout, want, err)
		}
		if _, err := s.AwaitURL(ctx, "https://x/await", in); err != nil || f.timeout != want {
			t.Fatalf("AwaitURL(%v): timeout %v, want %v (err %v)", in, f.timeout, want, err)
		}
	}
}

func TestPortErrorPassesThrough(t *testing.T) {
	want := errors.New("port failed")
	s := application.NewPayments(&fakeAPI{err: want})
	if _, err := s.GetPayment(ctx, "p1"); err != want {
		t.Fatalf("got %v", err)
	}
}
