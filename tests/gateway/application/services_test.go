package application_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/techpartners-asia/bonum-go/gateway/application"
	"github.com/techpartners-asia/bonum-go/gateway/domain"
	"github.com/techpartners-asia/bonum-go/gateway/domain/card"
	"github.com/techpartners-asia/bonum-go/gateway/domain/checkout"
	"github.com/techpartners-asia/bonum-go/gateway/domain/qr"
	"github.com/techpartners-asia/bonum-go/gateway/domain/subscription"
)

var errPort = errors.New("port failed")

func TestInvoicesValidateBeforePort(t *testing.T) {
	f := &fakeAPI{}
	s := application.NewInvoices(f)
	if _, err := s.Create(ctx, checkout.CreateInvoiceInput{}); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
	if len(f.calls) != 0 {
		t.Fatal("invalid input must not reach the port")
	}
	in := checkout.CreateInvoiceInput{Amount: 1, Callback: "https://m/cb", TransactionID: "T1", ExpiresIn: 60}
	inv, err := s.Create(ctx, in)
	if err != nil || inv.ID != "inv-1" || f.calls[0] != "CreateInvoice" || !reflect.DeepEqual(f.args[0], in) {
		t.Fatalf("forwarding: %v %v %v", inv, err, f.calls)
	}
	if p, err := s.Providers(ctx); err != nil || len(p) != 1 {
		t.Fatalf("providers: %v %v", p, err)
	}
}

func TestPortErrorsPassThroughUnwrapped(t *testing.T) {
	f := &fakeAPI{err: errPort}
	if _, err := application.NewInvoices(f).Providers(ctx); err != errPort {
		t.Fatalf("want the port error itself, got %v", err)
	}
	if err := application.NewCards(f).Reverse(ctx, "tok", "T1"); err != errPort {
		t.Fatalf("want the port error itself, got %v", err)
	}
}

func TestCards(t *testing.T) {
	f := &fakeAPI{}
	s := application.NewCards(f)
	if _, err := s.Tokenize(ctx, card.TokenizeInput{}); !errors.Is(err, domain.ErrInvalidInput) || len(f.calls) != 0 {
		t.Fatalf("tokenize validation: %v %v", err, f.calls)
	}
	if _, err := s.Purchase(ctx, "tok", card.PurchaseInput{}); !errors.Is(err, domain.ErrInvalidInput) || len(f.calls) != 0 {
		t.Fatalf("purchase validation: %v %v", err, f.calls)
	}
	in := card.PurchaseInput{Amount: 1, Currency: "MNT", TransactionID: "T1"}
	if p, err := s.Purchase(ctx, "tok", in); err != nil || p.ID != 1 || f.args[0] != "tok" || f.args[1] != in {
		t.Fatalf("purchase forwarding: %v %v %v", p, err, f.args)
	}
	if err := s.Reverse(ctx, "tok", "T1"); err != nil || f.calls[len(f.calls)-1] != "Reverse" {
		t.Fatalf("reverse: %v %v", err, f.calls)
	}
}

func TestSubscriptions(t *testing.T) {
	f := &fakeAPI{}
	s := application.NewSubscriptions(f)
	if _, err := s.Subscribe(ctx, "tok", subscription.SubscribeInput{}); !errors.Is(err, domain.ErrInvalidInput) || len(f.calls) != 0 {
		t.Fatalf("subscribe validation: %v", err)
	}
	if _, err := s.ChangeCardByTokenizing(ctx, 7, subscription.ChangeCardInput{}); !errors.Is(err, domain.ErrInvalidInput) || len(f.calls) != 0 {
		t.Fatalf("change card validation: %v", err)
	}
	if _, err := s.Subscribe(ctx, "tok", subscription.SubscribeInput{PlanID: 30, PayNow: true}); err != nil || f.calls[0] != "Subscribe" {
		t.Fatal(err)
	}
	if _, err := s.List(ctx, "tok"); err != nil || f.calls[1] != "ListSubscriptions" || f.args[0] != "tok" {
		t.Fatal(err)
	}
	if _, err := s.ChangeCard(ctx, 7, "tok"); err != nil || f.calls[2] != "ChangeCard" || f.args[0] != int64(7) {
		t.Fatal(err)
	}
	if err := s.Unsubscribe(ctx, 42, 30); err != nil || f.calls[3] != "Unsubscribe" || f.args[1] != int64(30) {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, 42, 30); err != nil || f.calls[4] != "DeleteSubscription" {
		t.Fatal(err)
	}
	if p, err := s.Plans(ctx); err != nil || len(p) != 1 {
		t.Fatal(err)
	}
}

func TestQR(t *testing.T) {
	f := &fakeAPI{}
	s := application.NewQR(f)
	if _, err := s.Create(ctx, qr.CreateQRInput{}); !errors.Is(err, domain.ErrInvalidInput) || len(f.calls) != 0 {
		t.Fatalf("create validation: %v", err)
	}
	if _, err := s.Lookup(ctx, ""); !errors.Is(err, domain.ErrInvalidInput) || len(f.calls) != 0 {
		t.Fatalf("lookup validation: %v", err)
	}
	if _, err := s.PayWithCard(ctx, "tok", qr.PayQRInput{}); !errors.Is(err, domain.ErrInvalidInput) || len(f.calls) != 0 {
		t.Fatalf("pay validation: %v", err)
	}
	if q, err := s.Lookup(ctx, "0002"); err != nil || q.QrCode != "0002" || f.calls[0] != "LookupQR" {
		t.Fatal(err)
	}
	if _, err := s.Create(ctx, qr.CreateQRInput{Amount: 1, TransactionID: "T1", ExpiresIn: 60}); err != nil || f.calls[1] != "CreateQR" {
		t.Fatal(err)
	}
	if p, err := s.PayWithCard(ctx, "tok", qr.PayQRInput{QrCode: "0002", TransactionID: "T1"}); err != nil || p.ID != 5 || f.args[0] != "tok" {
		t.Fatal(err)
	}
}

func TestSandboxAndAccessDelegate(t *testing.T) {
	f := &fakeAPI{}
	sb := application.NewSandbox(f)
	if raw, err := sb.InvoiceStatus(ctx, "inv-1"); err != nil || string(raw) != `{"status":"PAID"}` || f.args[0] != "inv-1" {
		t.Fatal(err)
	}
	if err := sb.MarkInvoicePaid(ctx, "inv-1"); err != nil || f.calls[1] != "MarkInvoicePaid" {
		t.Fatal(err)
	}
	if err := sb.RunSubscriptionBilling(ctx, 7); err != nil || f.calls[2] != "RunSubscriptionBilling" || f.args[0] != int64(7) {
		t.Fatal(err)
	}
	a := application.NewAccess(f)
	if tp, err := a.Authenticate(ctx); err != nil || tp.AccessToken != "a" || f.calls[3] != "CreateToken" {
		t.Fatal(err)
	}
	if tp, err := a.Refresh(ctx); err != nil || tp.AccessToken != "b" || f.calls[4] != "RefreshToken" {
		t.Fatal(err)
	}
}
