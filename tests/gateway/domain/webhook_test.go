package domain_test

import (
	"errors"
	"testing"

	"github.com/techpartners-asia/bonum-go/internal/gateway/domain/checkout"
	"github.com/techpartners-asia/bonum-go/internal/gateway/domain/webhook"
)

const key = "checksum-key"

func TestParseDispatchesOnType(t *testing.T) {
	body := []byte(`{"type":"PAYMENT","status":"SUCCESS","message":"ok","body":{"transactionId":"T1","amount":100,"currency":"MNT","terminalId":"17","invoiceId":"inv-1","paymentVendor":"QPAY","status":"PAID"}}`)
	ev, err := webhook.Parse(body, webhook.Checksum(body, key), key)
	if err != nil {
		t.Fatal(err)
	}
	p, ok := ev.(*webhook.PaymentEvent)
	if !ok || p.Outcome != webhook.OutcomeSuccess || p.Body.TransactionID != "T1" || *p.Body.PaymentVendor != checkout.ProviderQPay {
		t.Fatalf("unexpected %#v", ev)
	}
	if ev.Header().Type != webhook.EventPayment {
		t.Fatal("Header() must expose the envelope")
	}
}

func TestParseRejects(t *testing.T) {
	body := []byte(`{"type":"PAYMENT","status":"SUCCESS","body":{}}`)
	if _, err := webhook.Parse(body, "deadbeef", key); !errors.Is(err, webhook.ErrBadChecksum) {
		t.Fatalf("bad checksum: %v", err)
	}
	unknown := []byte(`{"type":"REFUND","status":"SUCCESS"}`)
	if _, err := webhook.Parse(unknown, webhook.Checksum(unknown, key), key); !errors.Is(err, webhook.ErrUnknownEvent) {
		t.Fatalf("unknown type: %v", err)
	}
	garbage := []byte(`nope`)
	if _, err := webhook.Parse(garbage, webhook.Checksum(garbage, key), key); err == nil {
		t.Fatal("garbage body must fail")
	}
}
