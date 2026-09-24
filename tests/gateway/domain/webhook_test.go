package domain_test

import (
	"errors"
	"strings"
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
	if _, err := webhook.Parse(body, "", key); !errors.Is(err, webhook.ErrBadChecksum) {
		t.Fatalf("empty header: %v", err)
	}
}

// Bonum signs JSON.toJson(body, prettyPrint = false), not the bytes on the wire. A delivery
// that arrives with whitespace must still verify against the compact form it was signed as.
func TestParseAcceptsWhitespaceAroundTheSignedCompactBody(t *testing.T) {
	signed := []byte(`{"type":"PAYMENT","status":"SUCCESS","body":{"amount":10000.00,"invoiceId":"inv-1","transactionId":"T1"}}`)
	sum := webhook.Checksum(signed, key)
	wire := []byte("{\n    \"type\": \"PAYMENT\",\n    \"status\": \"SUCCESS\",\n    \"body\": {\n        \"amount\": 10000.00,\n        \"invoiceId\": \"inv-1\",\n        \"transactionId\": \"T1\"\n    }\n}\n")

	ev, err := webhook.Parse(wire, sum, key)
	if err != nil {
		t.Fatalf("pretty-printed delivery of a compact-signed body: %v", err)
	}
	if p := ev.(*webhook.PaymentEvent); p.Body.TransactionID != "T1" || p.Body.Amount != 10000 {
		t.Fatalf("unexpected %#v", ev)
	}
	if _, err := webhook.Parse(wire, webhook.Checksum(signed, "other-key"), key); !errors.Is(err, webhook.ErrBadChecksum) {
		t.Fatalf("compact fallback must still require the key: %v", err)
	}
	tampered := []byte(`{"type":"PAYMENT","status":"SUCCESS","body":{"amount":99999.00,"invoiceId":"inv-1","transactionId":"T1"}}`)
	if _, err := webhook.Parse(tampered, sum, key); !errors.Is(err, webhook.ErrBadChecksum) {
		t.Fatalf("a changed value must not verify: %v", err)
	}
}

func TestParseComparesHexCaseInsensitively(t *testing.T) {
	body := []byte(`{"type":"PAYMENT","status":"SUCCESS","body":{"transactionId":"T1"}}`)
	if _, err := webhook.Parse(body, " "+strings.ToUpper(webhook.Checksum(body, key))+" ", key); err != nil {
		t.Fatalf("uppercase hex of the right MAC: %v", err)
	}
}
