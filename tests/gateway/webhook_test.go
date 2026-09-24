package gateway_test

import (
	"errors"
	"strings"
	"testing"

	bonum "github.com/techpartners-asia/bonum-go"
)

const (
	checksumKey = "test-checksum-key"
	paymentBody = `{"type":"PAYMENT","status":"SUCCESS","message":"","body":{"transactionId":"N998921","invoiceId":"8eff7d69001c03f486f64410f9daa82c","amount":10000.00,"currency":"MNT","paymentVendor":"QPAY","status":"PAID"}}`
	// Computed independently with: printf '%s' "$body" | openssl dgst -sha256 -hmac "$key"
	paymentChecksum = "fd2f3e0e2d0e0c1c9a7f1ff4b7fbe0a9e5b3d4c2a1f0e9d8c7b6a5f4e3d2c1b0"
)

func sign(body string) string { return bonum.Checksum([]byte(body), checksumKey) }

func TestParseWebhookRejectsBadChecksum(t *testing.T) {
	_, err := bonum.ParseWebhook([]byte(paymentBody), "deadbeef", checksumKey)
	if !errors.Is(err, bonum.ErrBadChecksum) {
		t.Fatalf("want ErrBadChecksum, got %v", err)
	}
	tampered := strings.Replace(paymentBody, `"amount":10000.00`, `"amount":10.00`, 1)
	_, err = bonum.ParseWebhook([]byte(tampered), sign(paymentBody), checksumKey)
	if !errors.Is(err, bonum.ErrBadChecksum) {
		t.Fatalf("tampered body: want ErrBadChecksum, got %v", err)
	}
	// Whitespace is not tampering: Bonum signs the compact serialisation, not the wire bytes.
	if _, err = bonum.ParseWebhook([]byte(paymentBody+"\n"), sign(paymentBody), checksumKey); err != nil {
		t.Fatalf("trailing newline on a compact-signed body: %v", err)
	}
}

func TestParseWebhookPayment(t *testing.T) {
	ev, err := bonum.ParseWebhook([]byte(paymentBody), sign(paymentBody), checksumKey)
	if err != nil {
		t.Fatal(err)
	}
	p, ok := ev.(*bonum.PaymentEvent)
	if !ok {
		t.Fatalf("want *PaymentEvent, got %T", ev)
	}
	if p.Outcome != bonum.OutcomeSuccess || p.Body.TransactionID != "N998921" || p.Body.Amount != 10000 {
		t.Fatalf("unexpected %+v", p)
	}
	if p.Body.InvoiceID == nil || *p.Body.InvoiceID != "8eff7d69001c03f486f64410f9daa82c" || *p.Body.PaymentVendor != bonum.ProviderQPay {
		t.Fatalf("success-only fields missing: %+v", p.Body)
	}
}

func TestParseWebhookPaymentFailed(t *testing.T) {
	body := `{"type":"PAYMENT","status":"FAILED","message":"expired","body":{"transactionId":"N1","amount":10,"currency":"MNT","updatedAt":1769660433000,"invoiceStatus":"EXPIRED"}}`
	ev, err := bonum.ParseWebhook([]byte(body), sign(body), checksumKey)
	if err != nil {
		t.Fatal(err)
	}
	p := ev.(*bonum.PaymentEvent)
	if p.Outcome != bonum.OutcomeFailed || p.Body.InvoiceStatus == nil || *p.Body.InvoiceStatus != "EXPIRED" || p.Body.InvoiceID != nil {
		t.Fatalf("unexpected %+v", p.Body)
	}
}

func TestParseWebhookCardToken(t *testing.T) {
	body := `{"type":"CARD-TOKEN","status":"SUCCESS","message":"","body":{"token":"tok-1","mask":"5150 23** **** 4778","expiry":"2026/11","bank":{"id":1,"code":"KHAN","name":"Khan Bank"},"transactionId":"T1","amounts":[{"amount":0.01,"currency":"MNT"}],"subscriptions":[{"subscriptionId":42,"planId":30,"nextBillingDate":"2026-03-01"}]}}`
	ev, err := bonum.ParseWebhook([]byte(body), sign(body), checksumKey)
	if err != nil {
		t.Fatal(err)
	}
	c := ev.(*bonum.CardTokenEvent)
	if c.Body.Token != "tok-1" || c.Body.Bank == nil || c.Body.Bank.Code != "KHAN" || c.Body.Subscriptions[0].SubscriptionID != 42 {
		t.Fatalf("unexpected %+v", c.Body)
	}
}

func TestParseWebhookTokenPaymentAndSubscription(t *testing.T) {
	body := `{"type":"TOKEN-PAYMENT","status":"SUCCESS","message":"","body":{"transactionId":"T9","completedAt":"2026-01-29 11:20:33"}}`
	ev, err := bonum.ParseWebhook([]byte(body), sign(body), checksumKey)
	if err != nil {
		t.Fatal(err)
	}
	if tp, ok := ev.(*bonum.TokenPaymentEvent); !ok || tp.Body.TransactionID != "T9" {
		t.Fatalf("unexpected %#v", ev)
	}
	body = `{"type":"SUBSCRIPTION-PAYMENT","status":"FAILED","message":"","body":{"subscriptionId":42,"invoiceId":7,"planId":30,"transactionId":"S1","amount":5000,"currency":"MNT"}}`
	ev, err = bonum.ParseWebhook([]byte(body), sign(body), checksumKey)
	if err != nil {
		t.Fatal(err)
	}
	if sp, ok := ev.(*bonum.SubscriptionPaymentEvent); !ok || sp.Body.SubscriptionID != 42 || sp.Outcome != bonum.OutcomeFailed {
		t.Fatalf("unexpected %#v", ev)
	}
}

func TestParseWebhookUnknownType(t *testing.T) {
	body := `{"type":"SOMETHING-NEW","status":"SUCCESS","message":"","body":{}}`
	_, err := bonum.ParseWebhook([]byte(body), sign(body), checksumKey)
	if !errors.Is(err, bonum.ErrUnknownEvent) {
		t.Fatalf("want ErrUnknownEvent, got %v", err)
	}
}

func TestChecksumMatchesOpenSSL(t *testing.T) {
	const (
		body = `{"type":"PAYMENT","status":"SUCCESS","message":"","body":{"transactionId":"N998921","invoiceId":"8eff7d69001c03f486f64410f9daa82c"}}`
		want = "bd77c3999d3cca578932fe9f1fd5b8e625885f9190b2490bc12f7610c39927be"
	)
	if got := bonum.Checksum([]byte(body), checksumKey); got != want {
		t.Fatalf("Checksum() = %s, want %s", got, want)
	}
}
