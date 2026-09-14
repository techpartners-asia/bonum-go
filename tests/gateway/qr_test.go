package gateway_test

import (
	"testing"

	bonum "github.com/techpartners-asia/bonum-go"
)

func TestQRCreateAndLookup(t *testing.T) {
	f, c := newFakeGateway(t, 1800)
	q, err := c.QR.Create(ctx, bonum.CreateQRInput{Amount: 500, TransactionID: "Q1", ExpiresIn: 300})
	if err != nil {
		t.Fatal(err)
	}
	if q.InvoiceID != "qr-1" || len(q.Links) != 1 || q.Links[0].Name != "Khan Bank" {
		t.Fatalf("unexpected %+v", q)
	}
	q2, err := c.QR.Lookup(ctx, "0002010102...")
	if err != nil {
		t.Fatal(err)
	}
	if q2.InvoiceID != "qr-1" || f.sentJSON(t)["qrCode"] != "0002010102..." {
		t.Fatalf("lookup wrong: %+v body=%s", q2, f.body())
	}
}

func TestQRPayWithCard(t *testing.T) {
	f, c := newFakeGateway(t, 1800)
	p, err := c.QR.PayWithCard(ctx, "card-tok", bonum.PayQRInput{QrCode: "0002", TransactionID: "Q2"})
	if err != nil {
		t.Fatal(err)
	}
	if p.ID != 5 || p.Status != bonum.PurchaseSuccess || f.lastReq.Load().Method != "PUT" {
		t.Fatalf("unexpected %+v", p)
	}
}
