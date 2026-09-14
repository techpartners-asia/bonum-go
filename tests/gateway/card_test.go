package gateway_test

import (
	"errors"
	"testing"

	bonum "github.com/techpartners-asia/bonum-go"
)

func TestCardsTokenize(t *testing.T) {
	_, c := newFakeGateway(t, 1800)
	tk, err := c.Cards.Tokenize(ctx, bonum.TokenizeInput{Callback: "https://m/cb", TransactionID: "T0"})
	if err != nil {
		t.Fatal(err)
	}
	if tk.ID != "tok-req-1" || tk.FollowUpLink == "" {
		t.Fatalf("unexpected %+v", tk)
	}
}

func TestCardsPurchaseUnwrapsEnvelope(t *testing.T) {
	f, c := newFakeGateway(t, 1800)
	p, err := c.Cards.Purchase(ctx, "card-tok", bonum.PurchaseInput{Amount: 15, Currency: "MNT", TransactionID: "T2"})
	if err != nil {
		t.Fatal(err)
	}
	if p.ID != 1 || p.Status != bonum.PurchaseSuccess || p.CardStatus == nil || *p.CardStatus != "ACTIVE" {
		t.Fatalf("unexpected purchase %+v", p)
	}
	if f.lastReq.Load().Header.Get("X-CARD-TOKEN") != "card-tok" {
		t.Fatal("X-CARD-TOKEN header missing")
	}
}

func TestCardsPurchaseQueued(t *testing.T) {
	_, c := newFakeGateway(t, 1800)
	p, err := c.Cards.Purchase(ctx, "queued-tok", bonum.PurchaseInput{Amount: 15, Currency: "MNT", TransactionID: "T2"})
	if err != nil {
		t.Fatal(err)
	}
	if p.Status != bonum.PurchaseQueued || p.CardStatus != nil {
		t.Fatalf("unexpected purchase %+v", p)
	}
}

func TestCardsPurchaseDeclinedIsTypedError(t *testing.T) {
	_, c := newFakeGateway(t, 1800)
	_, err := c.Cards.Purchase(ctx, "declined-tok", bonum.PurchaseInput{Amount: 15, Currency: "MNT", TransactionID: "T2"})
	if !errors.Is(err, bonum.ErrDeclined) {
		t.Fatalf("want ErrDeclined, got %v", err)
	}
	var d *bonum.DeclinedError
	if !errors.As(err, &d) || d.Purchase.ID != 9 || d.Purchase.Status != bonum.PurchaseFailed || d.TraceID != "t-d" {
		t.Fatalf("declined error not populated: %#v", err)
	}
	if !errors.Is(err, bonum.ErrInvalidInput) {
		t.Fatal("a decline is still a 400 and must satisfy ErrInvalidInput")
	}
}

func TestCardsPurchaseGeneric400IsNotDeclined(t *testing.T) {
	_, c := newFakeGateway(t, 1800)
	_, err := c.Cards.Purchase(ctx, "", bonum.PurchaseInput{Amount: 15, Currency: "MNT", TransactionID: "T3"})
	if !errors.Is(err, bonum.ErrInvalidInput) || errors.Is(err, bonum.ErrDeclined) {
		t.Fatalf("want plain ErrInvalidInput, got %v", err)
	}
	var api *bonum.APIError
	if !errors.As(err, &api) || api.TraceID != "t-1" || api.Message != "missing card token" {
		t.Fatalf("unexpected %#v", err)
	}
}

func TestCardsReverse(t *testing.T) {
	f, c := newFakeGateway(t, 1800)
	if err := c.Cards.Reverse(ctx, "card-tok", "txn-9"); err != nil {
		t.Fatal(err)
	}
	if f.lastReq.Load().Header.Get("X-CARD-TOKEN") != "card-tok" || f.lastReq.Load().Method != "DELETE" {
		t.Fatal("reverse must DELETE with the card token header")
	}
}
