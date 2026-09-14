package gateway_test

import (
	"errors"
	"testing"

	bonum "github.com/techpartners-asia/bonum-go"
)

func TestInvoicesCreateSendsBodyAndReturnsInvoice(t *testing.T) {
	f, c := newFakeGateway(t, 1800)
	inv, err := c.Invoices.Create(ctx, bonum.CreateInvoiceInput{
		Amount: 100, Callback: "https://m/cb", TransactionID: "T1", ExpiresIn: 600,
		Providers: []bonum.PaymentProvider{bonum.ProviderQPay},
	})
	if err != nil {
		t.Fatal(err)
	}
	if inv.ID != "inv-1" || inv.FollowUpLink == "" {
		t.Fatalf("unexpected invoice %+v", inv)
	}
	sent := f.sentJSON(t)
	if sent["transactionId"] != "T1" || sent["amount"].(float64) != 100 {
		t.Fatalf("unexpected body %v", sent)
	}
	if _, has := sent["items"]; has {
		t.Fatalf("empty optional items should be omitted: %v", sent)
	}
}

func TestInvoicesCreateValidatesBeforeCalling(t *testing.T) {
	f, c := newFakeGateway(t, 1800)
	_, err := c.Invoices.Create(ctx, bonum.CreateInvoiceInput{Amount: 0, Callback: "https://m/cb", TransactionID: "T1", ExpiresIn: 600})
	if !errors.Is(err, bonum.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
	if f.lastReq.Load() != nil {
		t.Fatal("invalid input must not reach the network")
	}
}

func TestInvoicesProviders(t *testing.T) {
	_, c := newFakeGateway(t, 1800)
	ps, err := c.Invoices.Providers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(ps) != 2 || ps[0].Provider != bonum.ProviderQPay || !ps[0].Enabled || ps[1].Enabled {
		t.Fatalf("unexpected providers %+v", ps)
	}
}

func TestSandboxHelpers(t *testing.T) {
	f, c := newFakeGateway(t, 1800)
	if err := c.Sandbox.MarkInvoicePaid(ctx, "inv-x"); err != nil {
		t.Fatal(err)
	}
	if got := f.lastReq.Load().URL.Query().Get("invoiceId"); got != "inv-x" {
		t.Fatalf("query param not forwarded: %q", got)
	}
	raw, err := c.Sandbox.InvoiceStatus(ctx, "inv-1")
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"invoiceId":"inv-1","status":"PAID"}` {
		t.Fatalf("raw = %s", raw)
	}
	if err := c.Sandbox.RunSubscriptionBilling(ctx, 7); err != nil {
		t.Fatal(err)
	}
}
